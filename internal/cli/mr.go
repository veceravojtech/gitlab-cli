package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/gitlab-cli/internal/config"
	"github.com/user/gitlab-cli/internal/gitlab"
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

var (
	listProject     int
	listMine        bool
	listApproved    bool
	showJSON        bool
	rebaseNoWait    bool
	mergeAutoRebase bool
	mergeMaxRetries int
	mergeTimeout    string

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
)

func init() {
	rootCmd.AddCommand(mrCmd)
	mrCmd.AddCommand(mrListCmd)
	mrCmd.AddCommand(mrShowCmd)
	mrCmd.AddCommand(mrRebaseCmd)
	mrCmd.AddCommand(mrMergeCmd)
	mrCmd.AddCommand(mrCreateCmd)

	mrListCmd.Flags().IntVar(&listProject, "project", 0, "filter by project ID")
	mrListCmd.Flags().BoolVar(&listMine, "mine", false, "only MRs assigned to me")
	mrListCmd.Flags().BoolVar(&listApproved, "approved", false, "only approved MRs")
	mrShowCmd.Flags().BoolVar(&showJSON, "json", false, "output as JSON")
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
	mrID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid MR ID: %s", args[0])
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	mr, err := client.GetMRByGlobalID(mrID)
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

	return nil
}

func runMRRebase(cmd *cobra.Command, args []string) error {
	mrID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid MR ID: %s", args[0])
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)
	prog := progress.New()

	mr, err := client.GetMRByGlobalID(mrID)
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
	mrID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid MR ID: %s", args[0])
	}

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
	deadline := time.Now().Add(timeout)
	attempt := 0

	prog := progress.New()

	// Get initial MR info for header
	mr, err := client.GetMRByGlobalID(mrID)
	if err != nil {
		return err
	}
	prog.Header("MR !%d: %s", mr.IID, mr.Title)

	var lastStatus string

	for {
		if time.Now().After(deadline) {
			prog.StopWait()
			return fmt.Errorf("timeout exceeded (%s)", mergeTimeout)
		}

		mr, err = client.GetMRByGlobalID(mrID)
		if err != nil {
			prog.StopWait()
			return err
		}

		// Only show status if changed
		if mr.DetailedMergeStatus != lastStatus {
			prog.StopWait()
			prog.Status(mr.DetailedMergeStatus)
			lastStatus = mr.DetailedMergeStatus
		}

		switch mr.DetailedMergeStatus {
		case "mergeable", "can_be_merged":
			prog.StopWait()
			prog.Action("Merging...")
			if err := client.MergeMR(mr.ProjectID, mr.IID); err != nil {
				if mergeAutoRebase && strings.Contains(err.Error(), "rebase") {
					prog.Action("Merge failed, needs rebase")
					continue
				}
				return err
			}
			prog.Success("MR merged successfully (%s total)", prog.TotalTime())
			return nil

		case "need_rebase", "cannot_be_merged_recheck":
			prog.StopWait()
			if !mergeAutoRebase {
				return fmt.Errorf("MR needs rebase. Run with --auto-rebase or: gitlab-cli mr rebase %d", mrID)
			}

			attempt++
			if attempt > mergeMaxRetries {
				return fmt.Errorf("max retries exceeded (%d)", mergeMaxRetries)
			}

			prog.Action("Triggering rebase... (attempt %d/%d)", attempt, mergeMaxRetries)
			if err := client.RebaseMR(mr.ProjectID, mr.IID); err != nil {
				return err
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

			prog.Action("Rebase complete")
			lastStatus = "" // Force status refresh

		case "conflict", "cannot_be_merged":
			prog.StopWait()
			prog.Error("Cannot merge: conflicts detected")
			return fmt.Errorf("cannot merge: conflicts detected\nResolve manually: %s", mr.WebURL)

		case "checking", "unchecked", "ci_still_running":
			statsFunc := func() string {
				if mr.HeadPipeline == nil {
					return ""
				}
				stats, err := client.GetPipelineStats(mr.ProjectID, mr.HeadPipeline.ID)
				if err != nil {
					return ""
				}
				return fmt.Sprintf("✓ %d passed | ● %d running | ○ %d pending",
					stats.Passed, stats.Running, stats.Pending)
			}
			prog.StartWait("Waiting for CI", statsFunc)
			time.Sleep(cfg.PollInterval)

		default:
			prog.StopWait()
			return fmt.Errorf("unexpected merge status: %s", mr.DetailedMergeStatus)
		}
	}
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
