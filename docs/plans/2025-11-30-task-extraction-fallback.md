# Task Extraction Fallback for Note Events Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract task IDs from `event.TargetTitle` when commenting on MRs/Issues where the task cannot be extracted from the branch name or MR title.

**Architecture:** Add a fallback in the Note event handling within `transformEvent()`. After existing extraction attempts fail (branch → MR title), try `extractTaskFromString(event.TargetTitle)` as final fallback.

**Tech Stack:** Go, existing `extractTaskFromString` function

---

### Task 1: Add Unit Tests for Task Extraction Functions

**Files:**
- Create: `internal/cli/activity_test.go`

**Step 1: Write failing tests for extractTaskFromString and extractTaskFromBranch**

```go
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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/... -run "TestExtract" -v`
Expected: FAIL - functions not exported (lowercase)

**Step 3: Export the extraction functions for testing**

In `internal/cli/activity.go`, the functions are already lowercase (unexported). For testing within the same package, they should work. Let's run the test again.

Run: `go test ./internal/cli/... -run "TestExtract" -v`
Expected: PASS (functions accessible within same package)

**Step 4: Commit**

```bash
git add internal/cli/activity_test.go
git commit -m "test(activity): add unit tests for task extraction functions"
```

---

### Task 2: Add Fallback to TargetTitle in Note Event Handling

**Files:**
- Modify: `internal/cli/activity.go:217-235`

**Step 1: Write failing test for TargetTitle fallback**

Add to `internal/cli/activity_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/... -run "TestTransformEvent" -v`
Expected: FAIL - task will be empty, not "#50607"

**Step 3: Implement the TargetTitle fallback**

In `internal/cli/activity.go`, modify the Note handling block (around line 217-235).

Change from:
```go
	case event.Note != nil:
		noteType := strings.ToLower(event.Note.NoteableType)
		description = fmt.Sprintf("commented on %s", noteType)
		if event.TargetTitle != "" {
			description += ": " + event.TargetTitle
		}
		details["noteable_type"] = noteType
		// If comment is on MR, get branch info and task
		if event.Note.NoteableType == "MergeRequest" {
			mr := getMRCached(event.ProjectID, event.TargetIID, mrCache, client)
			if mr != nil {
				source = mr.SourceBranch
				target = mr.TargetBranch
				task = extractTaskFromBranch(mr.SourceBranch)
				if task == "" {
					task = extractTaskFromString(mr.Title)
				}
			}
		}
```

To:
```go
	case event.Note != nil:
		noteType := strings.ToLower(event.Note.NoteableType)
		description = fmt.Sprintf("commented on %s", noteType)
		if event.TargetTitle != "" {
			description += ": " + event.TargetTitle
		}
		details["noteable_type"] = noteType
		// If comment is on MR, get branch info and task
		if event.Note.NoteableType == "MergeRequest" {
			mr := getMRCached(event.ProjectID, event.TargetIID, mrCache, client)
			if mr != nil {
				source = mr.SourceBranch
				target = mr.TargetBranch
				task = extractTaskFromBranch(mr.SourceBranch)
				if task == "" {
					task = extractTaskFromString(mr.Title)
				}
			}
		}
		// Fallback: extract task from TargetTitle if still unassigned
		if task == "" && event.TargetTitle != "" {
			task = extractTaskFromString(event.TargetTitle)
		}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/... -run "TestTransformEvent" -v`
Expected: PASS

**Step 5: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS

**Step 6: Commit**

```bash
git add internal/cli/activity.go internal/cli/activity_test.go
git commit -m "feat(activity): fallback to TargetTitle for task extraction in comments"
```

---

### Task 3: Manual Verification

**Step 1: Build and test with real data**

Run: `go build -o gitlab-cli . && ./gitlab-cli activity list --group-by-task 2>&1 | grep -A2 "commented on"`

Expected: "commented on" events should now appear under their correct task ID (e.g., `#50607`) instead of `Unassigned`

**Step 2: Verify no regressions in other output modes**

Run: `./gitlab-cli activity list | head -20`
Run: `./gitlab-cli activity list --json | head -50`

Expected: Output still works correctly, Task field populated where applicable

**Step 3: Final commit if any fixes needed**

Only if manual testing reveals issues.

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Unit tests for extraction functions | `internal/cli/activity_test.go` (create) |
| 2 | TargetTitle fallback implementation | `internal/cli/activity.go` (modify), `activity_test.go` (add tests) |
| 3 | Manual verification | N/A |
