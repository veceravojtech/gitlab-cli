package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/gitlab-cli/internal/config"
	"github.com/user/gitlab-cli/internal/gitlab"
)

var activityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Activity log operations",
}

var activityListCmd = &cobra.Command{
	Use:   "list",
	Short: "List user activities (defaults to current month)",
	RunE:  runActivityList,
}

var (
	activityPrev   bool
	activityFrom   string
	activityTo     string
	activityJSON   bool
	activityFormat string
)

func init() {
	rootCmd.AddCommand(activityCmd)
	activityCmd.AddCommand(activityListCmd)

	activityListCmd.Flags().BoolVar(&activityPrev, "prev", false, "show previous month")
	activityListCmd.Flags().StringVar(&activityFrom, "from", "", "start date (YYYY-MM-DD)")
	activityListCmd.Flags().StringVar(&activityTo, "to", "", "end date (YYYY-MM-DD)")
	activityListCmd.Flags().BoolVar(&activityJSON, "json", false, "output as JSON")
	activityListCmd.Flags().StringVar(&activityFormat, "format", "", "output format (csv)")
}

func getMonthRange(prev bool) (string, string) {
	now := time.Now()
	var year int
	var month time.Month

	if prev {
		year, month, _ = now.AddDate(0, -1, 0).Date()
	} else {
		year, month, _ = now.Date()
	}

	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	lastDay := firstDay.AddDate(0, 1, -1)

	return firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02")
}

func runActivityList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	// Determine date range
	var fromDate, toDate string
	if activityFrom != "" || activityTo != "" {
		fromDate = activityFrom
		toDate = activityTo
	} else {
		fromDate, toDate = getMonthRange(activityPrev)
	}

	// Fetch events
	events, err := client.GetEvents(gitlab.ListEventsOptions{
		After:  fromDate,
		Before: toDate,
	})
	if err != nil {
		return err
	}

	// Build project cache
	projectCache := make(map[int]string)
	for _, event := range events {
		if event.ProjectID > 0 {
			if _, ok := projectCache[event.ProjectID]; !ok {
				proj, err := client.GetProject(event.ProjectID)
				if err == nil {
					projectCache[event.ProjectID] = proj.Name
				} else {
					projectCache[event.ProjectID] = fmt.Sprintf("%d", event.ProjectID)
				}
			}
		}
	}

	// Transform to ActivityEntry
	activities := make([]gitlab.ActivityEntry, 0, len(events))
	for _, event := range events {
		entry := transformEvent(event, projectCache)
		activities = append(activities, entry)
	}

	// Output based on format
	if activityJSON {
		return outputJSON(activities)
	}
	if activityFormat == "csv" {
		return outputCSV(activities)
	}
	return outputTable(activities, fromDate, toDate)
}

func transformEvent(event gitlab.Event, projectCache map[int]string) gitlab.ActivityEntry {
	// Parse timestamp
	t, _ := time.Parse(time.RFC3339, event.CreatedAt)
	date := t.Format("2006-01-02")
	timeStr := t.Format("15:04")

	// Get project name
	projectName := ""
	if event.ProjectID > 0 {
		projectName = projectCache[event.ProjectID]
	}

	// Build description and details
	description := ""
	details := make(map[string]interface{})

	switch {
	case event.PushData != nil:
		pd := event.PushData
		if pd.CommitCount > 0 {
			description = fmt.Sprintf("%d commit(s) to %s", pd.CommitCount, pd.Ref)
			details["branch"] = pd.Ref
			details["commits"] = pd.CommitCount
		} else if pd.Action == "created" {
			description = fmt.Sprintf("created branch %s", pd.Ref)
			details["branch"] = pd.Ref
			details["action"] = "created"
		} else if pd.Action == "removed" {
			description = fmt.Sprintf("deleted branch %s", pd.Ref)
			details["branch"] = pd.Ref
			details["action"] = "deleted"
		}
	case event.TargetType == "MergeRequest":
		description = fmt.Sprintf("MR !%d: %s", event.TargetIID, event.TargetTitle)
		details["mr_iid"] = event.TargetIID
		details["title"] = event.TargetTitle
	case event.TargetType == "Issue":
		description = fmt.Sprintf("Issue #%d: %s", event.TargetIID, event.TargetTitle)
		details["issue_iid"] = event.TargetIID
		details["title"] = event.TargetTitle
	case event.Note != nil:
		noteType := strings.ToLower(event.Note.NoteableType)
		description = fmt.Sprintf("commented on %s", noteType)
		if event.TargetTitle != "" {
			description += ": " + event.TargetTitle
		}
		details["noteable_type"] = noteType
	default:
		if event.TargetTitle != "" {
			description = event.TargetTitle
		} else {
			description = event.ActionName
		}
	}

	return gitlab.ActivityEntry{
		Date:        date,
		Time:        timeStr,
		Type:        event.ActionName,
		Project:     projectName,
		Description: description,
		Details:     details,
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func outputTable(activities []gitlab.ActivityEntry, from, to string) error {
	fmt.Printf("Activity: %d events (%s to %s)\n\n", len(activities), from, to)

	if len(activities) == 0 {
		fmt.Println("No activities found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tTIME\tTYPE\tPROJECT\tDESCRIPTION")

	for _, a := range activities {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			a.Date,
			a.Time,
			a.Type,
			truncate(a.Project, 20),
			truncate(a.Description, 60),
		)
	}

	return w.Flush()
}

func outputCSV(activities []gitlab.ActivityEntry) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Header
	if err := w.Write([]string{"date", "time", "type", "project", "description"}); err != nil {
		return err
	}

	for _, a := range activities {
		if err := w.Write([]string{a.Date, a.Time, a.Type, a.Project, a.Description}); err != nil {
			return err
		}
	}

	return nil
}

func outputJSON(activities []gitlab.ActivityEntry) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(activities)
}
