package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/user/gitlab-cli/internal/gitlab"
)

// GetMRListWithCache returns the MR list, using cache if available and not disabled.
// Falls back to fresh API fetch on cache miss or when --no-cache flag is set.
func GetMRListWithCache(client *gitlab.Client) ([]gitlab.MergeRequest, error) {
	// Skip cache if --no-cache flag is set
	if !NoCacheEnabled() {
		cache, _ := LoadMRCache()
		if cache != nil && cache.IsValid() {
			return cache.MRs, nil
		}
	}

	// Fetch fresh from API (scope: assigned_to_me per epic design)
	mrs, err := client.ListMRs(gitlab.ListMROptions{Scope: "assigned_to_me"})
	if err != nil {
		return nil, fmt.Errorf("fetching MR list: %w", err)
	}

	// Save to cache (ignore save errors - non-critical)
	_ = SaveMRCache(mrs)

	return mrs, nil
}

// ResolutionResult holds the resolved MR information
type ResolutionResult struct {
	GlobalID  int    // The global MR ID for API calls
	IID       int    // Project-scoped IID for display (!3106)
	ProjectID int    // Project ID for API calls
	Title     string // MR title for confirmation display
	RawInput  string // Original user input (51706)
	MatchType string // Story 1.8: "" for IID match, "task#" for task number fallback
}

// MultiMatchError is returned when multiple MRs match the identifier
type MultiMatchError struct {
	Input   string                // Original input
	Matches []gitlab.MergeRequest // All matching MRs
}

func (e *MultiMatchError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ERROR: Multiple MRs match %s:\n", e.Input))
	for i, mr := range e.Matches {
		sb.WriteString(fmt.Sprintf("  [%d] !%d (project-%d) - %s\n", i+1, mr.IID, mr.ProjectID, mr.Title))
	}
	sb.WriteString("Use --select <index> to specify")
	return sb.String()
}

// containsTaskNumber checks if the title contains the task number with exact boundaries.
// The pattern #NNNNN must be followed by a word boundary (non-alphanumeric or end of string).
func containsTaskNumber(title string, taskNum int) bool {
	if title == "" {
		return false
	}
	// Pattern: #NNNNN followed by word boundary (non-alphanumeric or end of string)
	pattern := fmt.Sprintf(`#%d(?:[^a-zA-Z0-9]|$)`, taskNum)
	re := regexp.MustCompile(pattern)
	return re.MatchString(title)
}

// ResolveTaskNumber resolves a task number to a single MR from the provided list.
// Returns:
//   - ResolutionResult if exactly one MR matches
//   - error "No MR found matching..." if no MRs match
//   - MultiMatchError if multiple MRs match
func ResolveTaskNumber(taskNum int, rawInput string, mrs []gitlab.MergeRequest) (*ResolutionResult, error) {
	var matches []gitlab.MergeRequest

	for _, mr := range mrs {
		if containsTaskNumber(mr.Title, taskNum) {
			matches = append(matches, mr)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("No MR found matching %s in your assigned MRs", rawInput)
	case 1:
		mr := matches[0]
		return &ResolutionResult{
			GlobalID:  mr.ID,
			IID:       mr.IID,
			ProjectID: mr.ProjectID,
			Title:     mr.Title,
			RawInput:  rawInput,
		}, nil
	default:
		return nil, &MultiMatchError{
			Input:   rawInput,
			Matches: matches,
		}
	}
}

// ResolveIID resolves an IID to a single MR from the provided list.
// Returns:
//   - ResolutionResult if exactly one MR matches
//   - error "No MR found with IID..." if no MRs match
//   - MultiMatchError if multiple MRs match (different projects with same IID)
func ResolveIID(iid int, rawInput string, mrs []gitlab.MergeRequest) (*ResolutionResult, error) {
	var matches []gitlab.MergeRequest

	for _, mr := range mrs {
		if mr.IID == iid {
			matches = append(matches, mr)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("No MR found with IID %s in your assigned MRs", rawInput)
	case 1:
		mr := matches[0]
		return &ResolutionResult{
			GlobalID:  mr.ID,
			IID:       mr.IID,
			ProjectID: mr.ProjectID,
			Title:     mr.Title,
			RawInput:  rawInput,
		}, nil
	default:
		return nil, &MultiMatchError{
			Input:   rawInput,
			Matches: matches,
		}
	}
}

// globalIDFallbackThreshold is the value above which a numeric input might be
// a global ID rather than an IID. IIDs are typically < 10000 per project.
const globalIDFallbackThreshold = 10000

// NeedsGlobalIDFallback returns true if the value is large enough to possibly
// be a global ID rather than an IID. Used for backward compatibility.
func NeedsGlobalIDFallback(value int) bool {
	return value > globalIDFallbackThreshold
}

// ResolveUnified resolves a numeric identifier using priority-based matching:
// 1. IID match (exact IID column match)
// 2. Task# fallback (search for #VALUE in MR titles)
// Story 1.8: This is the core unified resolution function.
func ResolveUnified(value int, rawInput string, mrs []gitlab.MergeRequest) (*ResolutionResult, error) {
	// Priority 1: Try IID match
	result, err := ResolveIIDWithSelect(value, rawInput, mrs)
	if err == nil {
		return result, nil // IID match found
	}

	// If multiple IID matches, don't fallback - let user disambiguate
	if _, isMulti := err.(*MultiMatchError); isMulti {
		return nil, err
	}

	// Priority 2: Try Task# match (search for #VALUE in titles)
	result, err = ResolveTaskNumberWithSelect(value, rawInput, mrs)
	if err == nil {
		result.MatchType = "task#" // Mark for output formatting
		return result, nil
	}

	// Neither matched - return combined error message
	return nil, fmt.Errorf("No MR found with IID %d or task #%d in your assigned MRs", value, value)
}

// ResolveIdentifierWithMRs resolves a user-provided identifier using a pre-fetched MR list.
// This is the core resolution function that doesn't require a client.
// Story 1.8: Uses unified resolution - IID-first, task# fallback.
// For large numbers that don't match, caller should use the client to attempt
// global ID fallback using NeedsGlobalIDFallback() to check.
func ResolveIdentifierWithMRs(rawInput string, mrs []gitlab.MergeRequest) (*ResolutionResult, error) {
	parsed, err := ParseIdentifier(rawInput)
	if err != nil {
		return nil, err
	}

	// Story 1.8: All identifiers are now numeric - use unified resolution
	return ResolveUnified(parsed.Value, parsed.RawInput, mrs)
}

// ResolveIdentifier resolves a user-provided identifier to a ResolutionResult.
// Story 1.8: Uses unified resolution - IID-first, task# fallback.
// Falls back to global ID if resolution fails for large numbers.
func ResolveIdentifier(client *gitlab.Client, rawInput string) (*ResolutionResult, error) {
	parsed, err := ParseIdentifier(rawInput)
	if err != nil {
		return nil, err
	}

	mrs, err := GetMRListWithCache(client)
	if err != nil {
		return nil, fmt.Errorf("loading MR list: %w", err)
	}

	// Story 1.8: All identifiers are now numeric - use unified resolution
	result, err := ResolveUnified(parsed.Value, parsed.RawInput, mrs)
	if err != nil && NeedsGlobalIDFallback(parsed.Value) {
		// Fallback: large number might be a global ID
		return resolveGlobalIDFallback(client, parsed.Value, parsed.RawInput)
	}
	return result, err
}

// resolveGlobalIDFallback attempts to fetch MR by global ID directly.
func resolveGlobalIDFallback(client *gitlab.Client, globalID int, rawInput string) (*ResolutionResult, error) {
	mr, err := client.GetMRByGlobalID(globalID)
	if err != nil {
		return nil, fmt.Errorf("No MR found with ID %s", rawInput)
	}
	return &ResolutionResult{
		GlobalID:  mr.ID,
		IID:       mr.IID,
		ProjectID: mr.ProjectID,
		Title:     mr.Title,
		RawInput:  rawInput,
	}, nil
}
