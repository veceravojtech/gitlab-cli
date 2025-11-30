package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
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
	activityPrev        bool
	activityFrom        string
	activityTo          string
	activityJSON        bool
	activityFormat      string
	activityGroupByTask bool
)

func init() {
	rootCmd.AddCommand(activityCmd)
	activityCmd.AddCommand(activityListCmd)

	activityListCmd.Flags().BoolVar(&activityPrev, "prev", false, "show previous month")
	activityListCmd.Flags().StringVar(&activityFrom, "from", "", "start date (YYYY-MM-DD)")
	activityListCmd.Flags().StringVar(&activityTo, "to", "", "end date (YYYY-MM-DD)")
	activityListCmd.Flags().BoolVar(&activityJSON, "json", false, "output as JSON")
	activityListCmd.Flags().StringVar(&activityFormat, "format", "", "output format (csv)")
	activityListCmd.Flags().BoolVar(&activityGroupByTask, "group-by-task", false, "group activities by task")
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

	// Validate date format if provided
	if activityFrom != "" {
		if _, err := time.Parse("2006-01-02", activityFrom); err != nil {
			return fmt.Errorf("invalid --from date format, use YYYY-MM-DD")
		}
	}
	if activityTo != "" {
		if _, err := time.Parse("2006-01-02", activityTo); err != nil {
			return fmt.Errorf("invalid --to date format, use YYYY-MM-DD")
		}
	}

	// Fetch events
	events, err := client.GetEvents(gitlab.ListEventsOptions{
		After:  fromDate,
		Before: toDate,
	})
	if err != nil {
		return err
	}

	// Build project cache and default branch cache
	projectCache := make(map[int]string)
	defaultBranchCache := make(map[int]string)
	for _, event := range events {
		if event.ProjectID > 0 {
			if _, ok := projectCache[event.ProjectID]; !ok {
				proj, err := client.GetProject(event.ProjectID)
				if err == nil {
					projectCache[event.ProjectID] = proj.Name
					defaultBranchCache[event.ProjectID] = proj.DefaultBranch
				} else {
					projectCache[event.ProjectID] = fmt.Sprintf("%d", event.ProjectID)
					defaultBranchCache[event.ProjectID] = "main"
				}
			}
		}
	}

	// MR cache for branch lookups (key: "projectID-mrIID")
	mrCache := make(map[string]*gitlab.MergeRequest)

	// Transform to ActivityEntry
	activities := make([]gitlab.ActivityEntry, 0, len(events))
	for _, event := range events {
		entry := transformEvent(event, projectCache, defaultBranchCache, mrCache, client)
		activities = append(activities, entry)
	}

	// Output based on format
	if activityJSON {
		return outputJSON(activities)
	}
	if activityFormat == "csv" {
		return outputCSV(activities)
	}
	if activityGroupByTask {
		return outputGroupedTable(activities, fromDate, toDate)
	}
	return outputTable(activities, fromDate, toDate)
}

func transformEvent(event gitlab.Event, projectCache map[int]string, defaultBranchCache map[int]string, mrCache map[string]*gitlab.MergeRequest, client *gitlab.Client) gitlab.ActivityEntry {
	// Parse timestamp
	t, _ := time.Parse(time.RFC3339, event.CreatedAt)
	date := t.Format("2006-01-02")
	timeStr := t.Format("15:04")

	// Get project name
	projectName := ""
	if event.ProjectID > 0 {
		projectName = projectCache[event.ProjectID]
	}

	// Build description, details, and branch info
	description := ""
	details := make(map[string]interface{})
	var source, target, task string

	switch {
	case event.PushData != nil:
		pd := event.PushData
		source = pd.Ref
		details["branch"] = pd.Ref
		if pd.CommitCount > 0 {
			description = fmt.Sprintf("%d commit(s)", pd.CommitCount)
			details["commits"] = pd.CommitCount
		} else if pd.Action == "created" {
			description = "created branch"
			details["action"] = "created"
		} else if pd.Action == "removed" {
			description = "deleted branch"
			details["action"] = "deleted"
		}
		// Extract task from branch name first
		task = extractTaskFromBranch(pd.Ref)
		// If no task in branch and it's a default branch, try commit message
		if task == "" && pd.CommitCount > 0 {
			defaultBranch := defaultBranchCache[event.ProjectID]
			if pd.Ref == defaultBranch || pd.Ref == "main" || pd.Ref == "master" {
				// Fetch latest commit to extract task
				commits, err := client.GetCommits(event.ProjectID, pd.Ref, 1)
				if err == nil && len(commits) > 0 {
					task = extractTaskFromString(commits[0].Title)
					if task == "" {
						task = extractTaskFromString(commits[0].Message)
					}
				}
			}
		}
	case event.TargetType == "MergeRequest":
		mr := getMRCached(event.ProjectID, event.TargetIID, mrCache, client)
		if mr != nil {
			source = mr.SourceBranch
			target = mr.TargetBranch
			// Extract task from source branch first, then MR title
			task = extractTaskFromBranch(mr.SourceBranch)
			if task == "" {
				task = extractTaskFromString(mr.Title)
			}
		}
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
		// If comment is on MR, get branch info and task
		if event.Note.NoteableType == "MergeRequest" {
			mr := getMRCached(event.ProjectID, event.TargetIID, mrCache, client)
			if mr != nil {
				source = mr.SourceBranch
				target = mr.TargetBranch
				task = extractTaskFromBranch(mr.SourceBranch)
				if task == "" {
					task = extractTaskFromString(mr.Title)
				}
			}
		}
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
		Source:      source,
		Target:      target,
		Task:        task,
		Description: description,
		Details:     details,
	}
}

func getMRCached(projectID, mrIID int, cache map[string]*gitlab.MergeRequest, client *gitlab.Client) *gitlab.MergeRequest {
	key := fmt.Sprintf("%d-%d", projectID, mrIID)
	if mr, ok := cache[key]; ok {
		return mr
	}
	mr, err := client.GetMR(projectID, mrIID)
	if err != nil {
		cache[key] = nil
		return nil
	}
	cache[key] = mr
	return mr
}

var taskRegex = regexp.MustCompile(`#(\d{5})`)

func extractTaskFromBranch(branchName string) string {
	return extractTaskFromString(branchName)
}

func extractTaskFromString(s string) string {
	matches := taskRegex.FindStringSubmatch(s)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
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
	fmt.Fprintln(w, "DATE\tTIME\tTYPE\tPROJECT\tSOURCE\tTARGET\tTASK\tDESCRIPTION")

	for _, a := range activities {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			a.Date,
			a.Time,
			a.Type,
			truncate(a.Project, 20),
			truncate(a.Source, 18),
			truncate(a.Target, 12),
			a.Task,
			truncate(a.Description, 50),
		)
	}

	return w.Flush()
}

func outputCSV(activities []gitlab.ActivityEntry) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Header
	if err := w.Write([]string{"date", "time", "type", "project", "source", "target", "task", "description"}); err != nil {
		return err
	}

	for _, a := range activities {
		if err := w.Write([]string{a.Date, a.Time, a.Type, a.Project, a.Source, a.Target, a.Task, a.Description}); err != nil {
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

func outputGroupedTable(activities []gitlab.ActivityEntry, from, to string) error {
	fmt.Printf("Activity: %d events (%s to %s) - Grouped by Task\n\n", len(activities), from, to)

	if len(activities) == 0 {
		fmt.Println("No activities found")
		return nil
	}

	// Group by date, then by task
	type groupKey struct {
		date string
		task string
	}
	grouped := make(map[groupKey][]gitlab.ActivityEntry)
	dates := make(map[string]bool)
	tasks := make(map[string]bool)

	for _, a := range activities {
		task := a.Task
		if task == "" {
			task = "Unassigned"
		}
		key := groupKey{date: a.Date, task: task}
		grouped[key] = append(grouped[key], a)
		dates[a.Date] = true
		tasks[task] = true
	}

	// Sort dates descending
	sortedDates := make([]string, 0, len(dates))
	for d := range dates {
		sortedDates = append(sortedDates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(sortedDates)))

	// Sort tasks (Unassigned always last)
	sortedTasks := make([]string, 0, len(tasks))
	for t := range tasks {
		if t != "Unassigned" {
			sortedTasks = append(sortedTasks, t)
		}
	}
	sort.Strings(sortedTasks)
	if tasks["Unassigned"] {
		sortedTasks = append(sortedTasks, "Unassigned")
	}

	// Output grouped
	for _, date := range sortedDates {
		fmt.Printf("%s:\n", date)
		for _, task := range sortedTasks {
			key := groupKey{date: date, task: task}
			entries, ok := grouped[key]
			if !ok || len(entries) == 0 {
				continue
			}
			fmt.Printf("  %s:\n", task)
			for _, a := range entries {
				fmt.Printf("    - %s  %-12s  %-12s  %-18s  %s\n",
					a.Time,
					truncate(a.Type, 12),
					truncate(a.Project, 12),
					truncate(a.Source, 18),
					truncate(a.Description, 40),
				)
			}
		}
		fmt.Println()
	}

	return nil
}
