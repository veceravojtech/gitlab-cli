package cli

import (
	"testing"

	"github.com/user/gitlab-cli/internal/gitlab"
)

func TestExtractTaskFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"extracts task from MR title", "#50607: Fix login bug", "#50607"},
		{"extracts task from middle of string", "MR !123: #50607: Code review", "#50607"},
		{"returns empty for no task", "Random string without task", ""},
		{"returns empty for short numbers", "#1234 too short", ""},
		{"extracts exactly 5 digits", "#123456 six digits", "#12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTaskFromString(tt.input)
			if result != tt.expected {
				t.Errorf("extractTaskFromString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractTaskFromBranch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"extracts from feature branch", "feature/50607-fix-bug", "#50607"},
		{"extracts from plain branch", "50607-fix-bug", "#50607"},
		{"extracts with hash in branch", "feature/#50607-fix", "#50607"},
		{"returns empty for no task", "main", ""},
		{"returns empty for develop", "develop", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTaskFromBranch(tt.input)
			if result != tt.expected {
				t.Errorf("extractTaskFromBranch(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTransformEvent_NoteTargetTitleFallback(t *testing.T) {
	// Create mock event with Note where MR lookup would fail
	// but TargetTitle contains task ID
	event := gitlab.Event{
		ID:          1,
		ActionName:  "commented",
		CreatedAt:   "2025-11-25T16:13:00Z",
		ProjectID:   100,
		TargetType:  "MergeRequest",
		TargetIID:   123,
		TargetTitle: "#50607: Code review comments",
		Note: &gitlab.NoteData{
			NoteableType: "MergeRequest",
			Body:         "Some comment",
		},
	}

	projectCache := map[int]string{100: "TestProject"}
	defaultBranchCache := map[int]string{100: "main"}
	mrCache := make(map[string]*gitlab.MergeRequest)
	// No client - MR lookup will fail, forcing fallback

	entry := transformEvent(event, projectCache, defaultBranchCache, mrCache, nil)

	if entry.Task != "#50607" {
		t.Errorf("expected task #50607 from TargetTitle fallback, got %q", entry.Task)
	}
}

func TestTransformEvent_IssueCommentTargetTitleFallback(t *testing.T) {
	event := gitlab.Event{
		ID:          2,
		ActionName:  "commented",
		CreatedAt:   "2025-11-25T16:13:00Z",
		ProjectID:   100,
		TargetType:  "Issue",
		TargetIID:   456,
		TargetTitle: "#51234: Bug in checkout",
		Note: &gitlab.NoteData{
			NoteableType: "Issue",
			Body:         "Looking into this",
		},
	}

	projectCache := map[int]string{100: "TestProject"}
	defaultBranchCache := map[int]string{100: "main"}
	mrCache := make(map[string]*gitlab.MergeRequest)

	entry := transformEvent(event, projectCache, defaultBranchCache, mrCache, nil)

	if entry.Task != "#51234" {
		t.Errorf("expected task #51234 from Issue TargetTitle, got %q", entry.Task)
	}
}
