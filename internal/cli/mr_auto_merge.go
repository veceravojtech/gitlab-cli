package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/gitlab-cli/internal/config"
	"github.com/user/gitlab-cli/internal/gitlab"
)

func runMRAutoMerge(cmd *cobra.Command, args []string) error {
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

	if autoMergeCancel {
		if err := client.CancelAutoMerge(mr.ProjectID, mr.IID); err != nil {
			return err
		}
		fmt.Printf("Auto-merge cancelled for !%d\n", mr.IID)
		return nil
	}

	// Check if pipeline exists
	if mr.HeadPipeline == nil {
		return fmt.Errorf("no pipeline running, use 'mr merge' instead")
	}

	if mr.HasConflicts {
		return fmt.Errorf("MR has conflicts, resolve before enabling auto-merge")
	}

	if mr.State == "merged" {
		return fmt.Errorf("MR already merged")
	}

	if err := client.SetAutoMerge(mr.ProjectID, mr.IID); err != nil {
		return err
	}

	fmt.Printf("Auto-merge enabled for !%d\n", mr.IID)
	fmt.Printf("Will merge when pipeline succeeds (status: %s)\n", mr.HeadPipeline.Status)

	return nil
}
