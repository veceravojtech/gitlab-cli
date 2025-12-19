package cli

import (
	"fmt"
	"time"
)

// FormatResolutionOutput formats the resolution result for display.
// Story 1.8: Shows match type indicator for task# fallback matches.
// Output format: "Resolved: {input} → ID {GlobalID} (IID !{IID}, project-{ProjectID})"
// - IID match: "Resolved: 51706 → ID ..."
// - Task# fallback: "Resolved: 51706 (task#) → ID ..."
func FormatResolutionOutput(result *ResolutionResult) string {
	input := result.RawInput
	if result.MatchType == "task#" {
		input = fmt.Sprintf("%s (task#)", result.RawInput)
	}
	return fmt.Sprintf("Resolved: %s → ID %d (IID !%d, project-%d)",
		input,
		result.GlobalID,
		result.IID,
		result.ProjectID)
}

// FormatElapsedTime formats duration for user-friendly display.
// Format: "(Xs)" for < 60s, "(Xm Ys)" for >= 60s
func FormatElapsedTime(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	if totalSeconds < 60 {
		return fmt.Sprintf("(%ds)", totalSeconds)
	}
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("(%dm %ds)", minutes, seconds)
}

// PrintResolutionInfo prints resolution output to stdout.
func PrintResolutionInfo(result *ResolutionResult) {
	fmt.Println(FormatResolutionOutput(result))
}
