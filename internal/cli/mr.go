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

var (
	listProject     int
	listMine        bool
	showJSON        bool
	rebaseNoWait    bool
	mergeAutoRebase bool
	mergeMaxRetries int
	mergeTimeout    string
)

func init() {
	rootCmd.AddCommand(mrCmd)
	mrCmd.AddCommand(mrListCmd)
	mrCmd.AddCommand(mrShowCmd)
	mrCmd.AddCommand(mrRebaseCmd)
	mrCmd.AddCommand(mrMergeCmd)

	mrListCmd.Flags().IntVar(&listProject, "project", 0, "filter by project ID")
	mrListCmd.Flags().BoolVar(&listMine, "mine", false, "only MRs assigned to me")
	mrShowCmd.Flags().BoolVar(&showJSON, "json", false, "output as JSON")
	mrRebaseCmd.Flags().BoolVar(&rebaseNoWait, "no-wait", false, "don't wait for rebase to complete")
	mrMergeCmd.Flags().BoolVar(&mergeAutoRebase, "auto-rebase", false, "automatically rebase if needed")
	mrMergeCmd.Flags().IntVar(&mergeMaxRetries, "max-retries", 3, "max rebase attempts")
	mrMergeCmd.Flags().StringVar(&mergeTimeout, "timeout", "5m", "overall timeout")
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

	// Get MR to find project ID and IID
	mr, err := client.GetMRByGlobalID(mrID)
	if err != nil {
		return err
	}

	fmt.Printf("MR !%d: %s\n", mr.IID, mr.Title)
	fmt.Println("→ Triggering rebase...")

	if err := client.RebaseMR(mr.ProjectID, mr.IID); err != nil {
		return err
	}

	if rebaseNoWait {
		fmt.Println("Rebase triggered (not waiting for completion)")
		return nil
	}

	// Poll for rebase completion
	fmt.Print("→ Waiting for rebase...")

	for {
		time.Sleep(cfg.PollInterval)

		mr, err = client.GetMR(mr.ProjectID, mr.IID)
		if err != nil {
			return err
		}

		if !mr.RebaseInProgress {
			break
		}

		fmt.Print(".")
	}

	fmt.Println()

	if mr.MergeError != "" {
		return fmt.Errorf("rebase failed: %s", mr.MergeError)
	}

	fmt.Println("✓ Rebase complete")
	fmt.Printf("Status: %s\n", mr.DetailedMergeStatus)

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

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout exceeded (%s)", mergeTimeout)
		}

		// Get current MR status
		mr, err := client.GetMRByGlobalID(mrID)
		if err != nil {
			return err
		}

		fmt.Printf("MR !%d: %s\n", mr.IID, mr.Title)
		fmt.Printf("Status: %s\n", mr.DetailedMergeStatus)

		switch mr.DetailedMergeStatus {
		case "mergeable", "can_be_merged":
			fmt.Println("→ Merging...")
			if err := client.MergeMR(mr.ProjectID, mr.IID); err != nil {
				// Check if merge failed due to needing rebase
				if mergeAutoRebase && strings.Contains(err.Error(), "rebase") {
					fmt.Println("→ Merge failed, needs rebase")
					continue
				}
				return err
			}
			fmt.Println("✓ MR merged successfully")
			return nil

		case "need_rebase", "cannot_be_merged_recheck":
			if !mergeAutoRebase {
				return fmt.Errorf("MR needs rebase. Run with --auto-rebase or: gitlab-cli mr rebase %d", mrID)
			}

			attempt++
			if attempt > mergeMaxRetries {
				return fmt.Errorf("max retries exceeded (%d)", mergeMaxRetries)
			}

			fmt.Printf("→ Triggering rebase... (attempt %d/%d)\n", attempt, mergeMaxRetries)
			if err := client.RebaseMR(mr.ProjectID, mr.IID); err != nil {
				return err
			}

			// Wait for rebase to complete
			fmt.Print("→ Waiting for rebase...")
			for {
				time.Sleep(cfg.PollInterval)
				mr, err = client.GetMR(mr.ProjectID, mr.IID)
				if err != nil {
					return err
				}
				if !mr.RebaseInProgress {
					break
				}
				fmt.Print(".")
			}
			fmt.Println()

			if mr.MergeError != "" {
				return fmt.Errorf("rebase failed: %s", mr.MergeError)
			}

			fmt.Println("→ Rebase complete")
			fmt.Println("→ Checking merge status...")
			// Loop back to check status again

		case "conflict", "cannot_be_merged":
			return fmt.Errorf("cannot merge: conflicts detected\nResolve manually: %s", mr.WebURL)

		case "checking", "unchecked":
			fmt.Println("→ Waiting for CI/merge check...")
			time.Sleep(cfg.PollInterval)

		default:
			return fmt.Errorf("unexpected merge status: %s", mr.DetailedMergeStatus)
		}
	}
}
