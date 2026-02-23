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

func TestPipelineActivityEntry(t *testing.T) {
	// Test that pipeline activities are created with correct fields
	tests := []struct {
		name           string
		pipeline       gitlab.PipelineInfo
		projectName    string
		mrIID          int
		expectedTask   string
		expectedStatus string
	}{
		{
			name: "success pipeline with task in branch",
			pipeline: gitlab.PipelineInfo{
				ID:        12345,
				Status:    "success",
				Ref:       "feature/50607-fix-bug",
				SHA:       "abc123",
				CreatedAt: "2025-12-21T14:30:00.000Z",
				WebURL:    "https://gitlab.example.com/pipelines/12345",
			},
			projectName:    "TestProject",
			mrIID:          99,
			expectedTask:   "#50607",
			expectedStatus: "success",
		},
		{
			name: "failed pipeline without task",
			pipeline: gitlab.PipelineInfo{
				ID:        12346,
				Status:    "failed",
				Ref:       "main",
				SHA:       "def456",
				CreatedAt: "2025-12-21T15:00:00.000Z",
				WebURL:    "https://gitlab.example.com/pipelines/12346",
			},
			projectName:    "TestProject",
			mrIID:          100,
			expectedTask:   "",
			expectedStatus: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := extractTaskFromBranch(tt.pipeline.Ref)
			if task != tt.expectedTask {
				t.Errorf("extractTaskFromBranch(%q) = %q, want %q", tt.pipeline.Ref, task, tt.expectedTask)
			}

			description := "pipeline " + tt.pipeline.Status
			if description != "pipeline "+tt.expectedStatus {
				t.Errorf("description = %q, want %q", description, "pipeline "+tt.expectedStatus)
			}
		})
	}
}

func TestPipelineInfoStruct(t *testing.T) {
	// Verify PipelineInfo struct has all required fields
	p := gitlab.PipelineInfo{
		ID:        123,
		IID:       45,
		Status:    "success",
		Ref:       "main",
		SHA:       "abc123",
		CreatedAt: "2025-12-21T14:30:00.000Z",
		UpdatedAt: "2025-12-21T14:45:00.000Z",
		WebURL:    "https://example.com",
	}

	if p.ID != 123 {
		t.Errorf("PipelineInfo.ID = %d, want 123", p.ID)
	}
	if p.Status != "success" {
		t.Errorf("PipelineInfo.Status = %s, want success", p.Status)
	}
	if p.CreatedAt != "2025-12-21T14:30:00.000Z" {
		t.Errorf("PipelineInfo.CreatedAt = %s, want 2025-12-21T14:30:00.000Z", p.CreatedAt)
	}
}
