package mcp

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"ErrConfigLoad", ErrConfigLoad, "configuration load failed"},
		{"ErrConfigValidate", ErrConfigValidate, "configuration validation failed"},
		{"ErrAuthMissing", ErrAuthMissing, "authentication credentials missing"},
		{"ErrGitLabAPI", ErrGitLabAPI, "gitlab API error"},
		{"ErrMRNotFound", ErrMRNotFound, "merge request not found"},
		{"ErrProjectNotFound", ErrProjectNotFound, "project not found"},
		{"ErrInvalidInput", ErrInvalidInput, "invalid input"},
		{"ErrMissingParam", ErrMissingParam, "missing required parameter"},
		{"ErrMergeConflict", ErrMergeConflict, "merge conflict"},
		{"ErrMergeTimeout", ErrMergeTimeout, "merge timeout exceeded"},
		{"ErrRebaseFailed", ErrRebaseFailed, "rebase failed"},
		{"ErrPipelineFailed", ErrPipelineFailed, "pipeline failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("sentinel error is nil")
			}
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("got %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestWrappedErrors(t *testing.T) {
	tests := []struct {
		name        string
		wrapped     error
		target      error
		wantContain string
	}{
		{
			name:        "wrapped ErrMRNotFound",
			wrapped:     fmt.Errorf("%w: project 42 mr 7", ErrMRNotFound),
			target:      ErrMRNotFound,
			wantContain: "project 42 mr 7",
		},
		{
			name:        "wrapped ErrGitLabAPI",
			wrapped:     fmt.Errorf("%w: status 404", ErrGitLabAPI),
			target:      ErrGitLabAPI,
			wantContain: "status 404",
		},
		{
			name:        "wrapped ErrMergeConflict",
			wrapped:     fmt.Errorf("%w: resolve manually", ErrMergeConflict),
			target:      ErrMergeConflict,
			wantContain: "resolve manually",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.wrapped, tt.target) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.wrapped, tt.target)
			}
			if !strings.Contains(tt.wrapped.Error(), tt.wantContain) {
				t.Errorf("error %q does not contain %q", tt.wrapped.Error(), tt.wantContain)
			}
		})
	}
}

func TestErrorCategoriesDontCrossMatch(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		notErr error
	}{
		{"config vs api", ErrConfigLoad, ErrGitLabAPI},
		{"api vs validation", ErrGitLabAPI, ErrInvalidInput},
		{"validation vs merge", ErrInvalidInput, ErrMergeConflict},
		{"merge vs config", ErrMergeConflict, ErrConfigLoad},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errors.Is(tt.err, tt.notErr) {
				t.Errorf("errors.Is(%v, %v) = true, want false", tt.err, tt.notErr)
			}
		})
	}
}
