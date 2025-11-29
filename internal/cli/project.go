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

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Project operations",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List accessible projects",
	RunE:  runProjectList,
}

var (
	projectSearch     string
	projectOwned      bool
	projectMembership bool
	projectLimit      int
	projectJSON       bool
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectListCmd)

	projectListCmd.Flags().StringVar(&projectSearch, "search", "", "filter by project name")
	projectListCmd.Flags().BoolVar(&projectOwned, "owned", false, "only projects owned by me")
	projectListCmd.Flags().BoolVar(&projectMembership, "membership", true, "only projects I'm a member of")
	projectListCmd.Flags().IntVar(&projectLimit, "limit", 20, "number of results")
	projectListCmd.Flags().BoolVar(&projectJSON, "json", false, "output as JSON")
}

func runProjectList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	opts := gitlab.ListProjectsOptions{
		Search:     projectSearch,
		Owned:      projectOwned,
		Membership: projectMembership,
		PerPage:    projectLimit,
	}

	projects, err := client.ListProjects(opts)
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		fmt.Println("No projects found")
		return nil
	}

	if projectJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(projects)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPATH\tNAME\tVISIBILITY")

	for _, p := range projects {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			p.ID, p.PathWithNamespace, p.Name, p.Visibility)
	}

	w.Flush()
	return nil
}
