package cli

import "testing"

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
