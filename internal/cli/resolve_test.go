package cli

import (
	"strconv"
	"strings"
	"testing"

	"github.com/user/gitlab-cli/internal/gitlab"
)

func TestContainsTaskNumber(t *testing.T) {
	tests := []struct {
		title   string
		taskNum int
		want    bool
	}{
		{"#51706: Feature X", 51706, true},
		{"Fix for #51706", 51706, true},
		{"#51706", 51706, true},
		{"#51706 and #51707", 51706, true},
		{"Work on task #51706.", 51706, true},
		{"#516067", 51706, false},  // Different number (longer)
		{"#5170", 51706, false},    // Different number (shorter)
		{"51706", 51706, false},    // No # prefix
		{"", 51706, false},         // Empty string
		{"No task here", 51706, false},
		{"#51706x", 51706, false},  // Followed by letter (not word boundary)
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			if got := containsTaskNumber(tt.title, tt.taskNum); got != tt.want {
				t.Errorf("containsTaskNumber(%q, %d) = %v, want %v", tt.title, tt.taskNum, got, tt.want)
			}
		})
	}
}

func TestResolveTaskNumber(t *testing.T) {
	// Sample MR list for testing
	testMRs := []gitlab.MergeRequest{
		{ID: 14977, IID: 3106, ProjectID: 253, Title: "#51706: Feature X"},
		{ID: 14978, IID: 3107, ProjectID: 253, Title: "Unrelated MR"},
		{ID: 14979, IID: 892, ProjectID: 266, Title: "#51706: Hotfix Y"},
	}

	tests := []struct {
		name        string
		taskNum     int
		mrs         []gitlab.MergeRequest
		wantID      int
		wantErr     bool
		errContains string
	}{
		{
			name:    "single match",
			taskNum: 51707,
			mrs:     []gitlab.MergeRequest{{ID: 14980, IID: 100, ProjectID: 1, Title: "#51707: Solo"}},
			wantID:  14980,
			wantErr: false,
		},
		{
			name:        "no match",
			taskNum:     99999,
			mrs:         testMRs,
			wantErr:     true,
			errContains: "No MR found matching",
		},
		{
			name:        "multiple matches",
			taskNum:     51706,
			mrs:         testMRs,
			wantErr:     true,
			errContains: "Multiple MRs match",
		},
		{
			name:        "partial number no match",
			taskNum:     5170, // #5170 should not match #51706
			mrs:         testMRs,
			wantErr:     true,
			errContains: "No MR found matching",
		},
		{
			name:    "empty MR list",
			taskNum: 51706,
			mrs:     []gitlab.MergeRequest{},
			wantErr: true,
			errContains: "No MR found matching",
		},
		{
			name:    "task number in middle of title",
			taskNum: 12345,
			mrs:     []gitlab.MergeRequest{{ID: 100, IID: 10, ProjectID: 1, Title: "Fix for #12345 bug"}},
			wantID:  100,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawInput := "#" + strconv.Itoa(tt.taskNum)
			result, err := ResolveTaskNumber(tt.taskNum, rawInput, tt.mrs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveTaskNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if result.GlobalID != tt.wantID {
				t.Errorf("GlobalID = %d, want %d", result.GlobalID, tt.wantID)
			}
		})
	}
}

func TestMultiMatchErrorFormat(t *testing.T) {
	matches := []gitlab.MergeRequest{
		{ID: 14977, IID: 3106, ProjectID: 253, Title: "#51706: Feature X"},
		{ID: 14979, IID: 892, ProjectID: 266, Title: "#51706: Hotfix Y"},
	}

	err := &MultiMatchError{
		Input:   "#51706",
		Matches: matches,
	}

	errStr := err.Error()

	// Verify error message format per AC3
	if !strings.Contains(errStr, "ERROR: Multiple MRs match #51706") {
		t.Errorf("error should contain 'ERROR: Multiple MRs match #51706', got: %s", errStr)
	}
	if !strings.Contains(errStr, "[1]") {
		t.Errorf("error should contain '[1]', got: %s", errStr)
	}
	if !strings.Contains(errStr, "[2]") {
		t.Errorf("error should contain '[2]', got: %s", errStr)
	}
	if !strings.Contains(errStr, "!3106") {
		t.Errorf("error should contain '!3106', got: %s", errStr)
	}
	if !strings.Contains(errStr, "!892") {
		t.Errorf("error should contain '!892', got: %s", errStr)
	}
	if !strings.Contains(errStr, "--select") {
		t.Errorf("error should contain '--select', got: %s", errStr)
	}
}

func TestResolutionResultFields(t *testing.T) {
	mrs := []gitlab.MergeRequest{
		{ID: 14977, IID: 3106, ProjectID: 253, Title: "#51706: Feature X"},
	}

	result, err := ResolveTaskNumber(51706, "#51706", mrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.GlobalID != 14977 {
		t.Errorf("GlobalID = %d, want 14977", result.GlobalID)
	}
	if result.IID != 3106 {
		t.Errorf("IID = %d, want 3106", result.IID)
	}
	if result.ProjectID != 253 {
		t.Errorf("ProjectID = %d, want 253", result.ProjectID)
	}
	if result.Title != "#51706: Feature X" {
		t.Errorf("Title = %q, want %q", result.Title, "#51706: Feature X")
	}
	if result.RawInput != "#51706" {
		t.Errorf("RawInput = %q, want %q", result.RawInput, "#51706")
	}
}

func TestNoCacheEnabled(t *testing.T) {
	// Test that NoCacheEnabled returns a bool (function exists and is callable)
	// The actual value depends on flag state, but we verify it doesn't panic
	_ = NoCacheEnabled()
}

func TestResolveIID(t *testing.T) {
	// Sample MR list for testing - note: IID 3106 appears in two projects
	testMRs := []gitlab.MergeRequest{
		{ID: 14977, IID: 3106, ProjectID: 253, Title: "Feature X"},
		{ID: 14978, IID: 3107, ProjectID: 253, Title: "Unrelated MR"},
		{ID: 14979, IID: 3106, ProjectID: 266, Title: "Hotfix Y"}, // Same IID, different project!
		{ID: 14980, IID: 100, ProjectID: 1, Title: "Solo MR"},
	}

	tests := []struct {
		name        string
		iid         int
		mrs         []gitlab.MergeRequest
		wantID      int
		wantErr     bool
		errContains string
	}{
		{
			name:    "single match",
			iid:     100,
			mrs:     testMRs,
			wantID:  14980,
			wantErr: false,
		},
		{
			name:    "single match different IID",
			iid:     3107,
			mrs:     testMRs,
			wantID:  14978,
			wantErr: false,
		},
		{
			name:        "no match",
			iid:         99999,
			mrs:         testMRs,
			wantErr:     true,
			errContains: "No MR found with IID",
		},
		{
			name:        "multiple matches different projects",
			iid:         3106,
			mrs:         testMRs,
			wantErr:     true,
			errContains: "Multiple MRs match",
		},
		{
			name:        "partial IID no match",
			iid:         310, // Should NOT match IID 3106 or 3107
			mrs:         testMRs,
			wantErr:     true,
			errContains: "No MR found with IID",
		},
		{
			name:        "empty MR list",
			iid:         3106,
			mrs:         []gitlab.MergeRequest{},
			wantErr:     true,
			errContains: "No MR found with IID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveIID(tt.iid, strconv.Itoa(tt.iid), tt.mrs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveIID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if result.GlobalID != tt.wantID {
				t.Errorf("GlobalID = %d, want %d", result.GlobalID, tt.wantID)
			}
		})
	}
}

func TestResolveIIDResultFields(t *testing.T) {
	mrs := []gitlab.MergeRequest{
		{ID: 14980, IID: 100, ProjectID: 1, Title: "Solo MR"},
	}

	result, err := ResolveIID(100, "100", mrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.GlobalID != 14980 {
		t.Errorf("GlobalID = %d, want 14980", result.GlobalID)
	}
	if result.IID != 100 {
		t.Errorf("IID = %d, want 100", result.IID)
	}
	if result.ProjectID != 1 {
		t.Errorf("ProjectID = %d, want 1", result.ProjectID)
	}
	if result.Title != "Solo MR" {
		t.Errorf("Title = %q, want %q", result.Title, "Solo MR")
	}
	if result.RawInput != "100" {
		t.Errorf("RawInput = %q, want %q", result.RawInput, "100")
	}
}

func TestResolveIdentifierWithMRs(t *testing.T) {
	// Story 1.8: Test unified resolution - IID-first, task# fallback
	// Hash prefix (#NNNNN) is no longer supported
	testMRs := []gitlab.MergeRequest{
		{ID: 14977, IID: 3106, ProjectID: 253, Title: "#51706: Feature X"},
		{ID: 14978, IID: 3107, ProjectID: 253, Title: "Unrelated MR"},
		{ID: 14980, IID: 100, ProjectID: 1, Title: "Solo MR"},
	}

	tests := []struct {
		name        string
		input       string
		mrs         []gitlab.MergeRequest
		wantGlobal  int
		wantIID     int
		wantProject int
		wantErr     bool
		errContains string
	}{
		{
			name:        "IID resolution",
			input:       "3106",
			mrs:         testMRs,
			wantGlobal:  14977,
			wantIID:     3106,
			wantProject: 253,
			wantErr:     false,
		},
		{
			name:        "IID resolution different MR",
			input:       "100",
			mrs:         testMRs,
			wantGlobal:  14980,
			wantIID:     100,
			wantProject: 1,
			wantErr:     false,
		},
		{
			name:        "task number fallback - no IID match finds task# in title",
			input:       "51706",
			mrs:         testMRs,
			wantGlobal:  14977,
			wantIID:     3106,
			wantProject: 253,
			wantErr:     false,
		},
		{
			name:        "invalid format",
			input:       "abc123",
			mrs:         testMRs,
			wantErr:     true,
			errContains: "invalid identifier format",
		},
		{
			name:        "empty input",
			input:       "",
			mrs:         testMRs,
			wantErr:     true,
			errContains: "invalid identifier format",
		},
		{
			name:        "hash prefix no longer supported",
			input:       "#51706",
			mrs:         testMRs,
			wantErr:     true,
			errContains: "invalid identifier format",
		},
		{
			name:        "IID not found and no task# match",
			input:       "999",
			mrs:         testMRs,
			wantErr:     true,
			errContains: "No MR found with IID 999 or task #999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveIdentifierWithMRs(tt.input, tt.mrs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveIdentifierWithMRs(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if result.GlobalID != tt.wantGlobal {
				t.Errorf("GlobalID = %d, want %d", result.GlobalID, tt.wantGlobal)
			}
			if result.IID != tt.wantIID {
				t.Errorf("IID = %d, want %d", result.IID, tt.wantIID)
			}
			if result.ProjectID != tt.wantProject {
				t.Errorf("ProjectID = %d, want %d", result.ProjectID, tt.wantProject)
			}
			if result.RawInput != tt.input {
				t.Errorf("RawInput = %q, want %q", result.RawInput, tt.input)
			}
		})
	}
}

func TestResolveIdentifierWithMRsGlobalIDFallback(t *testing.T) {
	// Test the global ID fallback behavior for large numbers
	testMRs := []gitlab.MergeRequest{
		{ID: 14977, IID: 3106, ProjectID: 253, Title: "Feature X"},
	}

	// Large number with no IID match should indicate fallback needed
	// Note: The actual fallback requires a client call, so ResolveIdentifierWithMRs
	// returns a specific error that callers can check
	result, err := ResolveIdentifierWithMRs("14977", testMRs)

	// 14977 > 10000, so if no IID match is found, we need fallback
	// Since there's no IID 14977 in testMRs, this should fail
	// but the error should indicate that fallback might be needed
	if err == nil {
		// If somehow it resolved, check it didn't incorrectly match
		t.Errorf("Expected error for large IID with no match, got result: %+v", result)
	}
	if !strings.Contains(err.Error(), "No MR found with IID") {
		t.Errorf("Expected 'No MR found with IID' error, got: %v", err)
	}
}

// Story 1.8: Tests for ResolveUnified - priority-based resolution
func TestResolveUnified(t *testing.T) {
	tests := []struct {
		name          string
		value         int
		rawInput      string
		mrs           []gitlab.MergeRequest
		wantGlobalID  int
		wantMatchType string
		wantErr       bool
		errContains   string
	}{
		{
			name:     "IID match takes priority",
			value:    51706,
			rawInput: "51706",
			mrs: []gitlab.MergeRequest{
				{ID: 100, IID: 51706, ProjectID: 1, Title: "Some MR"},
				{ID: 200, IID: 999, ProjectID: 2, Title: "#51706: Feature"},
			},
			wantGlobalID:  100,
			wantMatchType: "", // IID match has empty MatchType
			wantErr:       false,
		},
		{
			name:     "task number fallback when no IID match",
			value:    51706,
			rawInput: "51706",
			mrs: []gitlab.MergeRequest{
				{ID: 200, IID: 999, ProjectID: 2, Title: "#51706: Feature"},
			},
			wantGlobalID:  200,
			wantMatchType: "task#",
			wantErr:       false,
		},
		{
			name:     "task number fallback with task in middle of title",
			value:    12345,
			rawInput: "12345",
			mrs: []gitlab.MergeRequest{
				{ID: 300, IID: 500, ProjectID: 1, Title: "Fix for #12345 bug"},
			},
			wantGlobalID:  300,
			wantMatchType: "task#",
			wantErr:       false,
		},
		{
			name:     "neither IID nor task# match",
			value:    99999,
			rawInput: "99999",
			mrs: []gitlab.MergeRequest{
				{ID: 100, IID: 3106, ProjectID: 1, Title: "Some MR"},
			},
			wantErr:     true,
			errContains: "No MR found with IID 99999 or task #99999",
		},
		{
			name:     "empty MR list",
			value:    51706,
			rawInput: "51706",
			mrs:      []gitlab.MergeRequest{},
			wantErr:  true,
			errContains: "No MR found with IID 51706 or task #51706",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveUnified(tt.value, tt.rawInput, tt.mrs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveUnified() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if result.GlobalID != tt.wantGlobalID {
				t.Errorf("GlobalID = %d, want %d", result.GlobalID, tt.wantGlobalID)
			}
			if result.MatchType != tt.wantMatchType {
				t.Errorf("MatchType = %q, want %q", result.MatchType, tt.wantMatchType)
			}
		})
	}
}

func TestResolveUnified_MultipleIIDMatchesNoFallback(t *testing.T) {
	// When multiple IIDs match, we should NOT fall back to task# resolution
	// User needs to use --select to disambiguate
	mrs := []gitlab.MergeRequest{
		{ID: 100, IID: 3106, ProjectID: 1, Title: "Feature in project 1"},
		{ID: 200, IID: 3106, ProjectID: 2, Title: "#3106: Same IID different project"},
	}

	_, err := ResolveUnified(3106, "3106", mrs)
	if err == nil {
		t.Error("expected MultiMatchError, got nil")
		return
	}

	// Should be MultiMatchError, not a regular error
	_, isMulti := err.(*MultiMatchError)
	if !isMulti {
		t.Errorf("expected *MultiMatchError, got %T", err)
	}

	// Error should mention multiple matches and --select
	if !strings.Contains(err.Error(), "Multiple MRs match") {
		t.Errorf("error should mention 'Multiple MRs match', got: %s", err.Error())
	}
}

func TestNeedsGlobalIDFallback(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  bool
	}{
		{"below threshold", 9999, false},
		{"at threshold", 10000, false},
		{"above threshold", 10001, true},
		{"typical global ID", 14977, true},
		{"very large", 100000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NeedsGlobalIDFallback(tt.value); got != tt.want {
				t.Errorf("NeedsGlobalIDFallback(%d) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
