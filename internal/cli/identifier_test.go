package cli

import (
	"strings"
	"testing"
)

func TestParseIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  IdentifierType
		wantValue int
		wantErr   bool
	}{
		// Story 1.8: Hash prefix is no longer supported - all must return error
		{"hash prefix rejected", "#51706", IdentifierTypeInvalid, 0, true},
		{"hash prefix zero rejected", "#0", IdentifierTypeInvalid, 0, true},
		{"hash prefix large rejected", "#999999", IdentifierTypeInvalid, 0, true},
		// Numeric identifiers work - resolved via unified IID-first, task# fallback
		{"iid", "3106", IdentifierTypeIID, 3106, false},
		{"large iid", "999999", IdentifierTypeIID, 999999, false},
		// Invalid formats
		{"invalid alpha", "abc123", IdentifierTypeInvalid, 0, true},
		{"invalid hash alpha", "#abc", IdentifierTypeInvalid, 0, true},
		{"empty string", "", IdentifierTypeInvalid, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIdentifier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIdentifier(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil {
				return // Error case, don't check values
			}
			if got.Type != tt.wantType {
				t.Errorf("ParseIdentifier(%q).Type = %v, want %v", tt.input, got.Type, tt.wantType)
			}
			if got.Value != tt.wantValue {
				t.Errorf("ParseIdentifier(%q).Value = %v, want %v", tt.input, got.Value, tt.wantValue)
			}
		})
	}
}

func TestParseIdentifier_ErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantErrContain string
	}{
		{"invalid alpha", "abc123", "invalid identifier format"},
		{"invalid hash alpha", "#abc", "invalid identifier format"},
		{"empty string", "", "invalid identifier format"},
		{"mixed format", "123abc", "invalid identifier format"},
		{"hash only", "#", "invalid identifier format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseIdentifier(tt.input)
			if err == nil {
				t.Errorf("ParseIdentifier(%q) expected error, got nil", tt.input)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("ParseIdentifier(%q) error = %q, want error containing %q", tt.input, err.Error(), tt.wantErrContain)
			}
		})
	}
}

func TestParseIdentifier_RawInputPreserved(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// Story 1.8: Only numeric identifiers - hash prefix no longer supported
		{"iid", "3106"},
		{"large iid", "51706"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIdentifier(tt.input)
			if err != nil {
				t.Fatalf("ParseIdentifier(%q) unexpected error: %v", tt.input, err)
			}
			if got.RawInput != tt.input {
				t.Errorf("ParseIdentifier(%q).RawInput = %q, want %q", tt.input, got.RawInput, tt.input)
			}
		})
	}
}
