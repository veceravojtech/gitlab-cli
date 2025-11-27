package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/gitlab-cli/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	RunE:  runConfigInit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("GitLab URL [https://gitlab.com]: ")
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)
	if url == "" {
		url = "https://gitlab.com"
	}

	fmt.Print("GitLab Token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token is required")
	}

	configPath := config.DefaultConfigPath()

	content := fmt.Sprintf(`gitlab_url: %s
gitlab_token: %s

defaults:
  max_retries: 3
  timeout: 5m
  poll_interval: 5s
`, url, token)

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Config written to %s\n", configPath)
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Printf("GitLab URL:    %s\n", cfg.GitLabURL)
	fmt.Printf("GitLab Token:  %s***\n", cfg.GitLabToken[:10])
	fmt.Printf("Max Retries:   %d\n", cfg.MaxRetries)
	fmt.Printf("Timeout:       %s\n", cfg.Timeout)
	fmt.Printf("Poll Interval: %s\n", cfg.PollInterval)

	return nil
}
