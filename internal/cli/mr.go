package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/gitlab-cli/internal/config"
	"github.com/user/gitlab-cli/internal/gitlab"
	"github.com/user/gitlab-cli/internal/mergeops"
	"github.com/user/gitlab-cli/internal/progress"
)

var mrCmd = &cobra.Command{
	Use:   "mr",
	Short: "Merge request operations",
}

var mrListCmd = &cobra.Command{
	Use:   "list",
	Short: "List open merge requests",
	RunE:  runMRList,
}

var mrShowCmd = &cobra.Command{
	Use:   "show <mr-id>",
	Short: "Show merge request details",
	Args:  cobra.ExactArgs(1),
	RunE:  runMRShow,
}

var mrRebaseCmd = &cobra.Command{
	Use:   "rebase <mr-id>",
	Short: "Rebase a merge request",
	Args:  cobra.ExactArgs(1),
	RunE:  runMRRebase,
}

var mrMergeCmd = &cobra.Command{
	Use:   "merge <mr-id>",
	Short: "Merge a merge request",
	Args:  cobra.ExactArgs(1),
	RunE:  runMRMerge,
}

var mrCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a merge request",
	RunE:  runMRCreate,
}

var mrLabelCmd = &cobra.Command{
	Use:   "label <mr-id>",
	Short: "Manage labels on a merge request",
	Args:  cobra.ExactArgs(1),
	RunE:  runMRLabel,
}

var mrAutoMergeCmd = &cobra.Command{
	Use:   "auto-merge <mr-id>",
	Short: "Enable or cancel merge when pipeline succeeds",
	Args:  cobra.ExactArgs(1),
	RunE:  runMRAutoMerge,
}

var mrReviewerCmd = &cobra.Command{
	Use:   "reviewer <mr-id>",
	Short: "Manage reviewers on a merge request",
	Args:  cobra.ExactArgs(1),
	RunE:  runMRReviewer,
}

var (
	listProject     int
	listMine        bool
	listApproved    bool
	showJSON        bool
	showDetail      bool
	showUnresolved  bool
	rebaseNoWait    bool
	mergeAutoRebase bool
	mergeMaxRetries int
	mergeTimeout    string
	noCacheFlag     bool
	selectIndex     int

	// mr create flags
	createProject            string
	createSource             string
	createTarget             string
	createTitle              string
	createDescription        string
	createDraft              bool
	createSquash             bool
	createRemoveSourceBranch bool
	createAllowCollab        bool
	createJSON               bool

	// mr label flags
	labelAdd    []string
	labelRemove []string
	labelList   bool

	// mr auto-merge flags
	autoMergeCancel bool

	// mr reviewer flags
	reviewerAdd    []string
	reviewerRemove []string
	reviewerList   bool
)

func init() {
	rootCmd.AddCommand(mrCmd)
	mrCmd.AddCommand(mrListCmd)
	mrCmd.AddCommand(mrShowCmd)
	mrCmd.AddCommand(mrRebaseCmd)
	mrCmd.AddCommand(mrMergeCmd)
	mrCmd.AddCommand(mrCreateCmd)
	mrCmd.AddCommand(mrLabelCmd)
	mrCmd.AddCommand(mrAutoMergeCmd)
	mrCmd.AddCommand(mrReviewerCmd)

	// Persistent flag for cache bypass - inherited by all MR subcommands
	mrCmd.PersistentFlags().BoolVar(&noCacheFlag, "no-cache", false, "bypass MR list cache")

	// Persistent flag for match selection - inherited by all MR subcommands
	mrCmd.PersistentFlags().IntVar(&selectIndex, "select", 0, "select match by index when multiple found")

	mrListCmd.Flags().IntVar(&listProject, "project", 0, "filter by project ID")
	mrListCmd.Flags().BoolVar(&listMine, "mine", false, "only MRs assigned to me")
	mrListCmd.Flags().BoolVar(&listApproved, "approved", false, "only approved MRs")
	mrShowCmd.Flags().BoolVar(&showJSON, "json", false, "output as JSON")
	mrShowCmd.Flags().BoolVar(&showDetail, "detail", false, "show full activity feed")
	mrShowCmd.Flags().BoolVar(&showUnresolved, "unresolved", false, "show only unresolved discussions (implies --detail)")
	mrRebaseCmd.Flags().BoolVar(&rebaseNoWait, "no-wait", false, "don't wait for rebase to complete")
	mrMergeCmd.Flags().BoolVar(&mergeAutoRebase, "auto-rebase", false, "automatically rebase if needed")
	mrMergeCmd.Flags().IntVar(&mergeMaxRetries, "max-retries", 3, "max rebase attempts")
	mrMergeCmd.Flags().StringVar(&mergeTimeout, "timeout", "5m", "overall timeout")

	mrCreateCmd.Flags().StringVar(&createProject, "project", "", "project ID or path (required)")
	mrCreateCmd.Flags().StringVar(&createSource, "source", "", "source branch (required)")
	mrCreateCmd.Flags().StringVar(&createTarget, "target", "", "target branch (required)")
	mrCreateCmd.Flags().StringVar(&createTitle, "title", "", "MR title (required)")
	mrCreateCmd.Flags().StringVar(&createDescription, "description", "", "MR description")
	mrCreateCmd.Flags().BoolVar(&createDraft, "draft", false, "create as draft MR")
	mrCreateCmd.Flags().BoolVar(&createSquash, "squash", false, "enable squash on merge")
	mrCreateCmd.Flags().BoolVar(&createRemoveSourceBranch, "remove-source-branch", false, "delete source branch after merge")
	mrCreateCmd.Flags().BoolVar(&createAllowCollab, "allow-collaboration", false, "allow commits from upstream members")
	mrCreateCmd.Flags().BoolVar(&createJSON, "json", false, "output as JSON")
	mrCreateCmd.MarkFlagRequired("project")
	mrCreateCmd.MarkFlagRequired("source")
	mrCreateCmd.MarkFlagRequired("target")
	mrCreateCmd.MarkFlagRequired("title")

	mrLabelCmd.Flags().StringSliceVar(&labelAdd, "add", nil, "add label (repeatable)")
	mrLabelCmd.Flags().StringSliceVar(&labelRemove, "remove", nil, "remove label (repeatable)")
	mrLabelCmd.Flags().BoolVar(&labelList, "list", false, "list current labels")

	mrAutoMergeCmd.Flags().BoolVar(&autoMergeCancel, "cancel", false, "cancel auto-merge")

	mrReviewerCmd.Flags().StringSliceVar(&reviewerAdd, "add", nil, "add reviewer by username or ID (repeatable)")
	mrReviewerCmd.Flags().StringSliceVar(&reviewerRemove, "remove", nil, "remove reviewer by username or ID (repeatable)")
	mrReviewerCmd.Flags().BoolVar(&reviewerList, "list", false, "list current reviewers")
}

func runMRList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	opts := gitlab.ListMROptions{
		State:     "opened",
		ProjectID: listProject,
	}

	if listMine {
		opts.Scope = "assigned_to_me"
	}

	if listApproved {
		opts.ApprovedByIDs = "Any"
	}

	mrs, err := client.ListMRs(opts)
	if err != nil {
		return err
	}

	if len(mrs) == 0 {
		fmt.Println("No open merge requests found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tIID\tPROJECT\tTITLE\tSTATUS")

	for _, mr := range mrs {
		title := mr.Title
		if len(title) > 45 {
			title = title[:42] + "..."
		}
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\n",
			mr.ID, mr.IID, mr.ProjectID, title, mr.DetailedMergeStatus)
	}

	w.Flush()
	return nil
}

func runMRShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	// Resolution layer: supports #NNNNN (task number), NNNNN (IID), and large numbers (global ID fallback)
	result, err := ResolveIdentifier(client, args[0])
	if err != nil {
		return err
	}
	PrintResolutionInfo(result)

	mr, err := client.GetMRByGlobalID(result.GlobalID)
	if err != nil {
		return err
	}

	if showJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(mr)
	}

	fmt.Printf("MR !%d: %s\n", mr.IID, mr.Title)
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Project:      %d\n", mr.ProjectID)
	fmt.Printf("Author:       %s\n", mr.Author.Name)
	fmt.Printf("Source:       %s → %s\n", mr.SourceBranch, mr.TargetBranch)
	fmt.Printf("Status:       %s\n", mr.DetailedMergeStatus)
	fmt.Printf("URL:          %s\n", mr.WebURL)

	// Show activity feed if --detail or --unresolved is specified
	if showDetail || showUnresolved {
		fmt.Println()
		fmt.Println("── Activity ──")
		if err := showMRActivity(client, mr, showUnresolved); err != nil {
			return err
		}
	}

	return nil
}

func runMRRebase(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	// Resolution layer: supports #NNNNN (task number), NNNNN (IID), and large numbers (global ID fallback)
	result, err := ResolveIdentifier(client, args[0])
	if err != nil {
		return err
	}
	PrintResolutionInfo(result)

	prog := progress.New()

	mr, err := client.GetMRByGlobalID(result.GlobalID)
	if err != nil {
		return err
	}

	prog.Header("MR !%d: %s", mr.IID, mr.Title)
	prog.Action("Triggering rebase...")

	if err := client.RebaseMR(mr.ProjectID, mr.IID); err != nil {
		return err
	}

	if rebaseNoWait {
		prog.Action("Rebase triggered (not waiting for completion)")
		return nil
	}

	prog.StartWait("Waiting for rebase", nil)

	for {
		time.Sleep(cfg.PollInterval)

		mr, err = client.GetMR(mr.ProjectID, mr.IID)
		if err != nil {
			prog.StopWait()
			return err
		}

		if !mr.RebaseInProgress {
			break
		}
	}

	prog.StopWait()

	if mr.MergeError != "" {
		prog.Error("Rebase failed: %s", mr.MergeError)
		return fmt.Errorf("rebase failed: %s", mr.MergeError)
	}

	prog.Success("Rebase complete (%s)", prog.TotalTime())
	prog.Status(mr.DetailedMergeStatus)

	return nil
}

func runMRMerge(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	timeout, err := time.ParseDuration(mergeTimeout)
	if err != nil {
		timeout = cfg.Timeout
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	// Resolution layer: supports #NNNNN (task number), NNNNN (IID), and large numbers (global ID fallback)
	result, err := ResolveIdentifier(client, args[0])
	if err != nil {
		return err
	}
	PrintResolutionInfo(result)

	prog := progress.New()

	// Get initial MR info for header
	mr, err := client.GetMRByGlobalID(result.GlobalID)
	if err != nil {
		return err
	}
	prog.Header("MR !%d: %s", mr.IID, mr.Title)

	// Set up context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	opts := mergeops.MergeOptions{
		ProjectID:    mr.ProjectID,
		MRIID:        mr.IID,
		AutoRebase:   mergeAutoRebase,
		MaxRetries:   mergeMaxRetries,
		Timeout:      timeout,
		PollInterval: cfg.PollInterval,
	}

	callback := func(status, detail string) {
		prog.StopWait()
		switch status {
		case "status":
			prog.Status(detail)
		case "merging", "rebasing", "rebase_needed", "rebase_complete":
			prog.Action(detail)
		case "waiting":
			prog.StartWait(detail, nil)
		default:
			prog.Action(detail)
		}
	}

	result2, mergeErr := mergeops.MergeWithRebase(ctx, client, opts, callback)

	prog.StopWait()

	if mergeErr != nil {
		prog.Error(mergeErr.Error())
		return mergeErr
	}

	_ = result2
	prog.Success("MR merged successfully (%s total)", prog.TotalTime())
	return nil
}

func runMRCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	opts := gitlab.CreateMROptions{
		SourceBranch:       createSource,
		TargetBranch:       createTarget,
		Title:              createTitle,
		Description:        createDescription,
		Draft:              createDraft,
		Squash:             createSquash,
		RemoveSourceBranch: createRemoveSourceBranch,
		AllowCollaboration: createAllowCollab,
	}

	mr, err := client.CreateMR(createProject, opts)
	if err != nil {
		return err
	}

	if createJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(mr)
	}

	fmt.Printf("Created MR !%d: %s\n", mr.IID, mr.Title)
	fmt.Printf("URL: %s\n", mr.WebURL)

	return nil
}

func runMRLabel(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	// Resolution layer: supports #NNNNN (task number), NNNNN (IID), and large numbers (global ID fallback)
	result, err := ResolveIdentifier(client, args[0])
	if err != nil {
		return err
	}
	PrintResolutionInfo(result)

	mr, err := client.GetMRByGlobalID(result.GlobalID)
	if err != nil {
		return err
	}

	// If no add/remove flags, just list labels
	if len(labelAdd) == 0 && len(labelRemove) == 0 {
		labelList = true
	}

	if labelList && len(labelAdd) == 0 && len(labelRemove) == 0 {
		fmt.Printf("Labels on !%d:\n", mr.IID)
		if len(mr.Labels) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, l := range mr.Labels {
				fmt.Printf("  • %s\n", l)
			}
		}
		return nil
	}

	// Compute new label set
	labelSet := make(map[string]bool)
	for _, l := range mr.Labels {
		labelSet[l] = true
	}

	for _, l := range labelAdd {
		labelSet[l] = true
	}

	for _, l := range labelRemove {
		delete(labelSet, l)
	}

	newLabels := make([]string, 0, len(labelSet))
	for l := range labelSet {
		newLabels = append(newLabels, l)
	}

	mr, err = client.UpdateMRLabels(mr.ProjectID, mr.IID, newLabels)
	if err != nil {
		return err
	}

	fmt.Printf("Labels on !%d:\n", mr.IID)
	if len(mr.Labels) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, l := range mr.Labels {
			fmt.Printf("  • %s\n", l)
		}
	}

	return nil
}

func runMRReviewer(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	// Resolution layer: supports #NNNNN (task number), NNNNN (IID), and large numbers (global ID fallback)
	result, err := ResolveIdentifier(client, args[0])
	if err != nil {
		return err
	}
	PrintResolutionInfo(result)

	mr, err := client.GetMRByGlobalID(result.GlobalID)
	if err != nil {
		return err
	}

	// If no add/remove flags, just list reviewers
	if len(reviewerAdd) == 0 && len(reviewerRemove) == 0 {
		reviewerList = true
	}

	if reviewerList && len(reviewerAdd) == 0 && len(reviewerRemove) == 0 {
		fmt.Printf("Reviewers on !%d:\n", mr.IID)
		if len(mr.Reviewers) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, r := range mr.Reviewers {
				fmt.Printf("  • %s (%s)\n", r.Username, r.Name)
			}
		}
		return nil
	}

	// Build current reviewer ID set
	reviewerSet := make(map[int]bool)
	for _, r := range mr.Reviewers {
		reviewerSet[r.ID] = true
	}

	// Resolve and add new reviewers
	for _, ref := range reviewerAdd {
		id, err := client.ResolveUserID(ref)
		if err != nil {
			return fmt.Errorf("resolving user '%s': %w", ref, err)
		}
		reviewerSet[id] = true
	}

	// Resolve and remove reviewers
	for _, ref := range reviewerRemove {
		id, err := client.ResolveUserID(ref)
		if err != nil {
			return fmt.Errorf("resolving user '%s': %w", ref, err)
		}
		delete(reviewerSet, id)
	}

	// Convert back to slice
	newReviewerIDs := make([]int, 0, len(reviewerSet))
	for id := range reviewerSet {
		newReviewerIDs = append(newReviewerIDs, id)
	}

	mr, err = client.UpdateMRReviewers(mr.ProjectID, mr.IID, newReviewerIDs)
	if err != nil {
		return err
	}

	fmt.Printf("Reviewers on !%d:\n", mr.IID)
	if len(mr.Reviewers) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, r := range mr.Reviewers {
			fmt.Printf("  • %s (%s)\n", r.Username, r.Name)
		}
	}

	return nil
}

func showMRActivity(client *gitlab.Client, mr *gitlab.MergeRequest, unresolvedOnly bool) error {
	// Fetch discussions
	discussions, err := client.GetMRDiscussions(mr.ProjectID, mr.IID)
	if err != nil {
		return err
	}

	// Fetch approvals
	approvals, err := client.GetMRApprovals(mr.ProjectID, mr.IID)
	if err != nil {
		// Non-fatal, some instances may not have approvals enabled
		approvals = &gitlab.ApprovalState{}
	}

	// Fetch label events
	labelEvents, err := client.GetMRLabelEvents(mr.ProjectID, mr.IID)
	if err != nil {
		// Non-fatal
		labelEvents = []gitlab.LabelEvent{}
	}

	// Collect all activities with timestamps for sorting
	type activity struct {
		timestamp string
		content   string
	}
	var activities []activity

	// Process discussions
	for _, d := range discussions {
		if len(d.Notes) == 0 {
			continue
		}

		// Check if discussion is resolved (for filtering)
		isResolved := false
		for _, n := range d.Notes {
			if n.Resolvable && n.Resolved {
				isResolved = true
				break
			}
		}

		if unresolvedOnly && isResolved {
			continue
		}

		// Skip system notes unless they're important
		firstNote := d.Notes[0]
		if firstNote.System {
			continue
		}

		var content strings.Builder

		// Format location if it's an inline comment
		location := ""
		if firstNote.Position != nil && firstNote.Position.NewPath != "" {
			location = fmt.Sprintf(" on %s:%d", firstNote.Position.NewPath, firstNote.Position.NewLine)
		}

		// First note
		content.WriteString(fmt.Sprintf("[%s] %s commented%s:\n",
			formatTimestamp(firstNote.CreatedAt),
			firstNote.Author.Username,
			location))
		content.WriteString(fmt.Sprintf("  %s\n", wrapText(firstNote.Body, 70)))

		// Replies
		for i, note := range d.Notes[1:] {
			if note.System {
				continue
			}
			prefix := "├─"
			if i == len(d.Notes)-2 {
				prefix = "└─"
			}
			content.WriteString(fmt.Sprintf("  %s [%s] %s replied:\n",
				prefix,
				formatTimestamp(note.CreatedAt),
				note.Author.Username))
			content.WriteString(fmt.Sprintf("  │    %s\n", wrapText(note.Body, 65)))
		}

		// Show resolved status
		if firstNote.Resolvable {
			if isResolved {
				content.WriteString("  └─ ✓ Resolved\n")
			} else {
				content.WriteString("  └─ ○ Unresolved\n")
			}
		}

		activities = append(activities, activity{
			timestamp: firstNote.CreatedAt,
			content:   content.String(),
		})
	}

	// Add approvals
	if approvals.Approved && len(approvals.Approvers) > 0 {
		for _, a := range approvals.Approvers {
			activities = append(activities, activity{
				timestamp: "2099-01-01", // Approvals don't have timestamp, show at end
				content:   fmt.Sprintf("[approved] %s approved this MR\n", a.User.Username),
			})
		}
	}

	// Add label events
	for _, e := range labelEvents {
		action := "added"
		if e.Action == "remove" {
			action = "removed"
		}
		activities = append(activities, activity{
			timestamp: e.CreatedAt,
			content:   fmt.Sprintf("[%s] label %s: %s\n", formatTimestamp(e.CreatedAt), action, e.Label.Name),
		})
	}

	// Sort by timestamp (simple string sort works for ISO timestamps)
	for i := 0; i < len(activities); i++ {
		for j := i + 1; j < len(activities); j++ {
			if activities[i].timestamp > activities[j].timestamp {
				activities[i], activities[j] = activities[j], activities[i]
			}
		}
	}

	// Print activities
	if len(activities) == 0 {
		fmt.Println("  (no activity)")
	} else {
		for _, a := range activities {
			fmt.Print(a.content)
			fmt.Println()
		}
	}

	return nil
}

func formatTimestamp(ts string) string {
	// Parse ISO timestamp and format nicely
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts[:16] // Fallback: just trim
	}
	return t.Format("2006-01-02 15:04")
}

func wrapText(text string, width int) string {
	// Simple text wrapper - just truncate long lines for now
	lines := strings.Split(text, "\n")
	if len(lines) > 3 {
		lines = lines[:3]
		lines = append(lines, "...")
	}
	return strings.Join(lines, "\n       ")
}
