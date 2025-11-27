package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("GITLAB_URL", "https://test.gitlab.com")
	os.Setenv("GITLAB_TOKEN", "test-token")
	defer os.Unsetenv("GITLAB_URL")
	defer os.Unsetenv("GITLAB_TOKEN")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GitLabURL != "https://test.gitlab.com" {
		t.Errorf("expected URL https://test.gitlab.com, got %s", cfg.GitLabURL)
	}
	if cfg.GitLabToken != "test-token" {
		t.Errorf("expected token test-token, got %s", cfg.GitLabToken)
	}
}
