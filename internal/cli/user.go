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

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User operations",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List and search users",
	RunE:  runUserList,
}

var (
	userSearch  string
	userProject string
	userLimit   int
	userJSON    bool
)

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userListCmd)

	userListCmd.Flags().StringVar(&userSearch, "search", "", "filter by name, username, or email")
	userListCmd.Flags().StringVar(&userProject, "project", "", "list only project members")
	userListCmd.Flags().IntVar(&userLimit, "limit", 20, "number of results")
	userListCmd.Flags().BoolVar(&userJSON, "json", false, "output as JSON")
}

func runUserList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

	var users []gitlab.User

	if userProject != "" {
		users, err = client.ListProjectMembers(userProject, userSearch)
	} else {
		users, err = client.ListUsers(gitlab.ListUsersOptions{
			Search:  userSearch,
			PerPage: userLimit,
		})
	}

	if err != nil {
		return err
	}

	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}

	if userJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(users)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUSERNAME\tNAME\tEMAIL")

	for _, u := range users {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			u.ID, u.Username, u.Name, u.Email)
	}

	w.Flush()
	return nil
}
