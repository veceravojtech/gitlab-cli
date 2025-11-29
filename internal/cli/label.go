package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/user/gitlab-cli/internal/config"
	"github.com/user/gitlab-cli/internal/gitlab"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Label operations",
}

var labelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List project labels",
	RunE:  runLabelList,
}

var (
	labelProject string
	labelSearch  string
	labelJSON    bool
)

func init() {
	rootCmd.AddCommand(labelCmd)
	labelCmd.AddCommand(labelListCmd)

	labelListCmd.Flags().StringVar(&labelProject, "project", "", "project ID or path (required)")
	labelListCmd.Flags().StringVar(&labelSearch, "search", "", "filter labels by name")
	labelListCmd.Flags().BoolVar(&labelJSON, "json", false, "output as JSON")
	labelListCmd.MarkFlagRequired("project")
}

func runLabelList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	labels, err := client.ListProjectLabels(labelProject, labelSearch)
	if err != nil {
		return err
	}

	if len(labels) == 0 {
		fmt.Println("No labels found")
		return nil
	}

	if labelJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(labels)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCOLOR\tDESCRIPTION")

	for _, l := range labels {
		desc := l.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", l.Name, l.Color, desc)
	}

	w.Flush()
	return nil
}
