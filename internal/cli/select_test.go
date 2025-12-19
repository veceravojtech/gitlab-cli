package cli

import (
	"strings"
	"testing"

	"github.com/user/gitlab-cli/internal/gitlab"
)

func TestResolveWithSelect(t *testing.T) {
	// Sample MR list with multiple matches (same scenario as MultiMatchError)
	testMatches := []gitlab.MergeRequest{
		{ID: 14977, IID: 3106, ProjectID: 253, Title: "#51706: Feature X"},
		{ID: 14979, IID: 892, ProjectID: 266, Title: "#51706: Hotfix Y"},
	}

	tests := []struct {
		name        string
		matches     []gitlab.MergeRequest
		selectIdx   int
		rawInput    string
		wantID      int
		wantErr     bool
		errContains string
	}{
		{
			name:      "select first match",
			matches:   testMatches,
			selectIdx: 1,
			rawInput:  "#51706",
			wantID:    14977,
			wantErr:   false,
		},
		{
			name:      "select second match",
			matches:   testMatches,
			selectIdx: 2,
			rawInput:  "#51706",
			wantID:    14979,
			wantErr:   false,
		},
		{
			name:        "out of range - too high",
			matches:     testMatches,
			selectIdx:   5,
			rawInput:    "#51706",
			wantErr:     true,
			errContains: "Invalid selection: 5. Only 2 matches found.",
		},
		{
			name:        "out of range - zero",
			matches:     testMatches,
			selectIdx:   0,
			rawInput:    "#51706",
			wantErr:     true,
			errContains: "Selection must be >= 1",
		},
		{
			name:        "out of range - negative",
			matches:     testMatches,
			selectIdx:   -1,
			rawInput:    "#51706",
			wantErr:     true,
			errContains: "Selection must be >= 1",
		},
		{
			name:      "single match - select 1",
			matches:   testMatches[:1], // Only first MR
			selectIdx: 1,
			rawInput:  "#51706",
			wantID:    14977,
			wantErr:   false,
		},
		{
			name:        "empty matches",
			matches:     []gitlab.MergeRequest{},
			selectIdx:   1,
			rawInput:    "#51706",
			wantErr:     true,
			errContains: "Only 0 matches found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveWithSelect(tt.matches, tt.selectIdx, tt.rawInput)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveWithSelect() error = %v, wantErr %v", err, tt.wantErr)
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

func TestSelectHelpers(t *testing.T) {
	tests := []struct {
		name         string
		selectIdx    int
		wantEnabled  bool
		wantGetIndex int
	}{
		{"zero - disabled", 0, false, 0},
		{"positive - enabled", 3, true, 3},
		{"one - enabled", 1, true, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original and restore after test
			original := selectIndex
			defer func() { selectIndex = original }()

			selectIndex = tt.selectIdx

			if got := SelectEnabled(); got != tt.wantEnabled {
				t.Errorf("SelectEnabled() = %v, want %v", got, tt.wantEnabled)
			}
			if got := GetSelectIndex(); got != tt.wantGetIndex {
				t.Errorf("GetSelectIndex() = %v, want %v", got, tt.wantGetIndex)
			}
		})
	}
}

func TestResolveTaskNumberWithSelect(t *testing.T) {
	// MRs with same task number in title (produces MultiMatchError)
	multiMatchMRs := []gitlab.MergeRequest{
		{ID: 14977, IID: 3106, ProjectID: 253, Title: "#51706: Feature X"},
		{ID: 14979, IID: 892, ProjectID: 266, Title: "#51706: Hotfix Y"},
	}

	// MR with unique task number
	singleMatchMRs := []gitlab.MergeRequest{
		{ID: 14977, IID: 3106, ProjectID: 253, Title: "#51706: Feature X"},
		{ID: 14980, IID: 3107, ProjectID: 253, Title: "#99999: Other MR"},
	}

	tests := []struct {
		name        string
		taskNum     int
		rawInput    string
		mrs         []gitlab.MergeRequest
		selectIdx   int
		wantID      int
		wantErr     bool
		errContains string
	}{
		{
			name:      "single match - no select needed",
			taskNum:   51706,
			rawInput:  "#51706",
			mrs:       singleMatchMRs,
			selectIdx: 0,
			wantID:    14977,
			wantErr:   false,
		},
		{
			name:      "single match - select ignored",
			taskNum:   51706,
			rawInput:  "#51706",
			mrs:       singleMatchMRs,
			selectIdx: 2,
			wantID:    14977,
			wantErr:   false,
		},
		{
			name:        "multiple matches - no select",
			taskNum:     51706,
			rawInput:    "#51706",
			mrs:         multiMatchMRs,
			selectIdx:   0,
			wantErr:     true,
			errContains: "Multiple MRs match",
		},
		{
			name:      "multiple matches - select 1",
			taskNum:   51706,
			rawInput:  "#51706",
			mrs:       multiMatchMRs,
			selectIdx: 1,
			wantID:    14977,
			wantErr:   false,
		},
		{
			name:      "multiple matches - select 2",
			taskNum:   51706,
			rawInput:  "#51706",
			mrs:       multiMatchMRs,
			selectIdx: 2,
			wantID:    14979,
			wantErr:   false,
		},
		{
			name:        "multiple matches - select out of range",
			taskNum:     51706,
			rawInput:    "#51706",
			mrs:         multiMatchMRs,
			selectIdx:   5,
			wantErr:     true,
			errContains: "Invalid selection: 5. Only 2 matches found.",
		},
		{
			name:        "no match - select doesn't help",
			taskNum:     88888,
			rawInput:    "#88888",
			mrs:         multiMatchMRs,
			selectIdx:   1,
			wantErr:     true,
			errContains: "No MR found matching",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore selectIndex
			original := selectIndex
			defer func() { selectIndex = original }()
			selectIndex = tt.selectIdx

			result, err := ResolveTaskNumberWithSelect(tt.taskNum, tt.rawInput, tt.mrs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveTaskNumberWithSelect() error = %v, wantErr %v", err, tt.wantErr)
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

func TestResolveIIDWithSelect(t *testing.T) {
	// MRs with same IID in different projects (produces MultiMatchError)
	multiMatchMRs := []gitlab.MergeRequest{
		{ID: 14977, IID: 100, ProjectID: 253, Title: "MR in Project A"},
		{ID: 14979, IID: 100, ProjectID: 266, Title: "MR in Project B"},
	}

	// MR with unique IID
	singleMatchMRs := []gitlab.MergeRequest{
		{ID: 14977, IID: 100, ProjectID: 253, Title: "MR in Project A"},
		{ID: 14980, IID: 200, ProjectID: 253, Title: "Different IID"},
	}

	tests := []struct {
		name        string
		iid         int
		rawInput    string
		mrs         []gitlab.MergeRequest
		selectIdx   int
		wantID      int
		wantErr     bool
		errContains string
	}{
		{
			name:      "single match - no select needed",
			iid:       100,
			rawInput:  "!100",
			mrs:       singleMatchMRs,
			selectIdx: 0,
			wantID:    14977,
			wantErr:   false,
		},
		{
			name:      "single match - select ignored",
			iid:       100,
			rawInput:  "!100",
			mrs:       singleMatchMRs,
			selectIdx: 2,
			wantID:    14977,
			wantErr:   false,
		},
		{
			name:        "multiple matches - no select",
			iid:         100,
			rawInput:    "!100",
			mrs:         multiMatchMRs,
			selectIdx:   0,
			wantErr:     true,
			errContains: "Multiple MRs match",
		},
		{
			name:      "multiple matches - select 1",
			iid:       100,
			rawInput:  "!100",
			mrs:       multiMatchMRs,
			selectIdx: 1,
			wantID:    14977,
			wantErr:   false,
		},
		{
			name:      "multiple matches - select 2",
			iid:       100,
			rawInput:  "!100",
			mrs:       multiMatchMRs,
			selectIdx: 2,
			wantID:    14979,
			wantErr:   false,
		},
		{
			name:        "multiple matches - select out of range",
			iid:         100,
			rawInput:    "!100",
			mrs:         multiMatchMRs,
			selectIdx:   5,
			wantErr:     true,
			errContains: "Invalid selection: 5. Only 2 matches found.",
		},
		{
			name:        "no match - select doesn't help",
			iid:         999,
			rawInput:    "!999",
			mrs:         multiMatchMRs,
			selectIdx:   1,
			wantErr:     true,
			errContains: "No MR found with IID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore selectIndex
			original := selectIndex
			defer func() { selectIndex = original }()
			selectIndex = tt.selectIdx

			result, err := ResolveIIDWithSelect(tt.iid, tt.rawInput, tt.mrs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveIIDWithSelect() error = %v, wantErr %v", err, tt.wantErr)
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
