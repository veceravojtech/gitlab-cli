package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	GitLabURL    string
	GitLabToken  string
	MaxRetries   int
	Timeout      time.Duration
	PollInterval time.Duration
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("max_retries", 3)
	v.SetDefault("timeout", "5m")
	v.SetDefault("poll_interval", "5s")

	// Environment variables
	v.SetEnvPrefix("")
	v.BindEnv("gitlab_url", "GITLAB_URL")
	v.BindEnv("gitlab_token", "GITLAB_TOKEN")
	v.BindEnv("max_retries", "GITLAB_CLI_MAX_RETRIES")
	v.BindEnv("timeout", "GITLAB_CLI_TIMEOUT")
	v.AutomaticEnv()

	// Config file
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(home)
			v.SetConfigName(".gitlab-cli")
			v.SetConfigType("yaml")
		}
	}

	// Read config file (ignore if not found)
	v.ReadInConfig()

	timeout, err := time.ParseDuration(v.GetString("timeout"))
	if err != nil {
		timeout = 5 * time.Minute
	}

	pollInterval, err := time.ParseDuration(v.GetString("poll_interval"))
	if err != nil {
		pollInterval = 5 * time.Second
	}

	cfg := &Config{
		GitLabURL:    v.GetString("gitlab_url"),
		GitLabToken:  v.GetString("gitlab_token"),
		MaxRetries:   v.GetInt("max_retries"),
		Timeout:      timeout,
		PollInterval: pollInterval,
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.GitLabURL == "" {
		return fmt.Errorf("gitlab_url is required")
	}
	if c.GitLabToken == "" {
		return fmt.Errorf("gitlab_token is required")
	}
	return nil
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gitlab-cli.yaml")
}
