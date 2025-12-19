package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	numericRegex = regexp.MustCompile(`^(\d+)$`)
)

// IdentifierType represents the type of MR identifier provided by the user
type IdentifierType int

const (
	IdentifierTypeInvalid IdentifierType = iota
	IdentifierTypeIID                    // NNNNN - resolved via unified IID-first, task# fallback
)

// ParsedIdentifier holds the result of parsing a user-provided MR identifier
type ParsedIdentifier struct {
	Type     IdentifierType
	Value    int    // The numeric value extracted
	RawInput string // Original input for error messages
}

// ParseIdentifier parses a user-provided MR identifier and returns its type and value.
// Story 1.8: Only numeric format is supported. Hash prefix (#NNNNN) is no longer valid.
// Resolution uses unified IID-first, task# fallback strategy.
func ParseIdentifier(input string) (*ParsedIdentifier, error) {
	if input == "" {
		return nil, fmt.Errorf("invalid identifier format: %s", input)
	}

	// Story 1.8: Reject hash prefix - breaking change per epic design
	if strings.HasPrefix(input, "#") {
		return nil, fmt.Errorf("invalid identifier format: %s", input)
	}

	// Only numeric format allowed
	if matches := numericRegex.FindStringSubmatch(input); matches != nil {
		value, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("parsing identifier %q: %w", input, err)
		}
		return &ParsedIdentifier{
			Type:     IdentifierTypeIID,
			Value:    value,
			RawInput: input,
		}, nil
	}

	return nil, fmt.Errorf("invalid identifier format: %s", input)
}
