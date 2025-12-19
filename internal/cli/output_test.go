package cli

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/user/gitlab-cli/internal/gitlab"
)

func TestFormatResolutionOutput(t *testing.T) {
	tests := []struct {
		name   string
		result *ResolutionResult
		want   string
	}{
		{
			name: "IID resolution - no match type indicator",
			result: &ResolutionResult{
				GlobalID:  14977,
				IID:       3106,
				ProjectID: 253,
				Title:     "Feature X",
				RawInput:  "3106",
				MatchType: "", // IID match has empty MatchType
			},
			want: "Resolved: 3106 → ID 14977 (IID !3106, project-253)",
		},
		{
			name: "task number fallback - shows (task#) indicator",
			result: &ResolutionResult{
				GlobalID:  14977,
				IID:       3106,
				ProjectID: 253,
				Title:     "#51706: Feature X",
				RawInput:  "51706",
				MatchType: "task#", // Task number match shows indicator
			},
			want: "Resolved: 51706 (task#) → ID 14977 (IID !3106, project-253)",
		},
		{
			name: "different project - IID match",
			result: &ResolutionResult{
				GlobalID:  14979,
				IID:       892,
				ProjectID: 266,
				Title:     "Hotfix Y",
				RawInput:  "892",
				MatchType: "",
			},
			want: "Resolved: 892 → ID 14979 (IID !892, project-266)",
		},
		{
			name: "task number in different project",
			result: &ResolutionResult{
				GlobalID:  14979,
				IID:       892,
				ProjectID: 266,
				Title:     "#51706: Hotfix Y",
				RawInput:  "51706",
				MatchType: "task#",
			},
			want: "Resolved: 51706 (task#) → ID 14979 (IID !892, project-266)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatResolutionOutput(tt.result)
			if got != tt.want {
				t.Errorf("FormatResolutionOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatElapsedTime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "0 seconds",
			duration: 0 * time.Second,
			want:     "(0s)",
		},
		{
			name:     "42 seconds",
			duration: 42 * time.Second,
			want:     "(42s)",
		},
		{
			name:     "59 seconds",
			duration: 59 * time.Second,
			want:     "(59s)",
		},
		{
			name:     "exactly 60 seconds",
			duration: 60 * time.Second,
			want:     "(1m 0s)",
		},
		{
			name:     "2 minutes 15 seconds",
			duration: 135 * time.Second,
			want:     "(2m 15s)",
		},
		{
			name:     "5 minutes exactly",
			duration: 5 * time.Minute,
			want:     "(5m 0s)",
		},
		{
			name:     "10 minutes 30 seconds",
			duration: 10*time.Minute + 30*time.Second,
			want:     "(10m 30s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatElapsedTime(tt.duration)
			if got != tt.want {
				t.Errorf("FormatElapsedTime(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestPrintResolutionInfo(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := &ResolutionResult{
		GlobalID:  14977,
		IID:       3106,
		ProjectID: 253,
		Title:     "#51706: Feature X",
		RawInput:  "#51706",
	}

	PrintResolutionInfo(result)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	got := buf.String()

	want := "Resolved: #51706 → ID 14977 (IID !3106, project-253)\n"
	if got != want {
		t.Errorf("PrintResolutionInfo() output = %q, want %q", got, want)
	}
}

func TestResolutionOutputFlow(t *testing.T) {
	// Integration test: Resolution → Output flow
	// This simulates what Story 1.7 will do when integrating with MR commands

	// Given: An MR list and task number input
	mrs := []gitlab.MergeRequest{
		{
			ID:        14977,
			IID:       3106,
			ProjectID: 253,
			Title:     "#51706: Feature X",
		},
	}
	rawInput := "#51706"
	taskNum := 51706

	// When: Resolution succeeds
	result, err := ResolveTaskNumber(taskNum, rawInput, mrs)
	if err != nil {
		t.Fatalf("ResolveTaskNumber() error = %v", err)
	}

	// Then: Output format matches AC1 exactly
	output := FormatResolutionOutput(result)
	want := "Resolved: #51706 → ID 14977 (IID !3106, project-253)"
	if output != want {
		t.Errorf("Full flow output = %q, want %q", output, want)
	}
}

func TestIIDResolutionOutputFlow(t *testing.T) {
	// Integration test: IID Resolution → Output flow

	// Given: An MR list and IID input
	mrs := []gitlab.MergeRequest{
		{
			ID:        14977,
			IID:       3106,
			ProjectID: 253,
			Title:     "Feature X",
		},
	}
	rawInput := "3106"
	iid := 3106

	// When: Resolution succeeds
	result, err := ResolveIID(iid, rawInput, mrs)
	if err != nil {
		t.Fatalf("ResolveIID() error = %v", err)
	}

	// Then: Output format matches AC1 exactly
	output := FormatResolutionOutput(result)
	want := "Resolved: 3106 → ID 14977 (IID !3106, project-253)"
	if output != want {
		t.Errorf("Full flow output = %q, want %q", output, want)
	}
}
