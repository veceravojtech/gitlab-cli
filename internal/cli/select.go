package cli

import (
	"errors"
	"fmt"

	"github.com/user/gitlab-cli/internal/gitlab"
)

// SelectEnabled returns true when the --select flag is set to a positive value.
func SelectEnabled() bool {
	return selectIndex > 0
}

// GetSelectIndex returns the current value of the --select flag.
func GetSelectIndex() int {
	return selectIndex
}

// ResolveWithSelect selects a specific MR from a list of matches by index.
// The selectIdx is 1-indexed (human-friendly).
// Returns:
//   - ResolutionResult if valid index is provided
//   - error if index is out of range or invalid
func ResolveWithSelect(matches []gitlab.MergeRequest, selectIdx int, rawInput string) (*ResolutionResult, error) {
	// Validate selection index
	if selectIdx < 1 {
		return nil, fmt.Errorf("invalid selection: %d. Selection must be >= 1", selectIdx)
	}
	if selectIdx > len(matches) {
		return nil, fmt.Errorf("Invalid selection: %d. Only %d matches found.", selectIdx, len(matches))
	}

	// Select the match (convert to 0-indexed)
	mr := matches[selectIdx-1]
	return &ResolutionResult{
		GlobalID:  mr.ID,
		IID:       mr.IID,
		ProjectID: mr.ProjectID,
		Title:     mr.Title,
		RawInput:  rawInput,
	}, nil
}

// ResolveTaskNumberWithSelect wraps ResolveTaskNumber with --select flag handling.
// If ResolveTaskNumber returns MultiMatchError and SelectEnabled() is true,
// it attempts to resolve using the select index.
func ResolveTaskNumberWithSelect(taskNum int, rawInput string, mrs []gitlab.MergeRequest) (*ResolutionResult, error) {
	result, err := ResolveTaskNumber(taskNum, rawInput, mrs)
	if err == nil {
		return result, nil
	}

	// Check if it's a MultiMatchError and selection is enabled
	var multiErr *MultiMatchError
	if errors.As(err, &multiErr) && SelectEnabled() {
		return ResolveWithSelect(multiErr.Matches, GetSelectIndex(), rawInput)
	}

	return nil, err
}

// ResolveIIDWithSelect wraps ResolveIID with --select flag handling.
// If ResolveIID returns MultiMatchError and SelectEnabled() is true,
// it attempts to resolve using the select index.
func ResolveIIDWithSelect(iid int, rawInput string, mrs []gitlab.MergeRequest) (*ResolutionResult, error) {
	result, err := ResolveIID(iid, rawInput, mrs)
	if err == nil {
		return result, nil
	}

	// Check if it's a MultiMatchError and selection is enabled
	var multiErr *MultiMatchError
	if errors.As(err, &multiErr) && SelectEnabled() {
		return ResolveWithSelect(multiErr.Matches, GetSelectIndex(), rawInput)
	}

	return nil, err
}
