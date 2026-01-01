package internal_test

import (
	"errors"
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
)

func TestCleanNamespacedIdentifier(t *testing.T) {
	// Simulate quoting requirements of different DBs
	wrapBackticks := func(s string) string { return "`" + s + "`" }
	wrapDoubleQuotes := func(s string) string { return "\"" + s + "\"" }

	tests := []struct {
		name        string
		input       string
		quotePart   func(string) string
		expected    string
		expectError bool
	}{
		// OK cases
		{
			name:      "single table name, mysql style quote wrapper",
			input:     "users",
			quotePart: wrapBackticks,
			expected:  "`users`",
		},
		{
			name:      "namespaced table, postgres style quote wrapper",
			input:     "public.users",
			quotePart: wrapDoubleQuotes,
			expected:  `"public"."users"`,
		},
		{
			name:      "input with existing quotes plus some normalization",
			input:     "`foo`.`bar` ",
			quotePart: wrapBackticks,
			expected:  "`foo`.`bar`",
		},

		// Error cases
		{
			name:        "too many dots",
			input:       "too.many.dots",
			quotePart:   wrapDoubleQuotes,
			expectError: true,
		},
		{
			name:        "sql injection, comment",
			input:       `foobars; --`,
			quotePart:   wrapBackticks,
			expectError: true,
		},
		{
			name:        "sql injection, query",
			input:       `foobars' OR '1'='1`,
			quotePart:   wrapDoubleQuotes,
			expectError: true,
		},
		{
			name:        "empty",
			input:       "",
			quotePart:   wrapBackticks,
			expectError: true,
		},
		{
			name:        "starts with number",
			input:       "123bad",
			quotePart:   wrapBackticks,
			expectError: true,
		},
		{
			name:        "invalid characters that are not stripped away",
			input:       "foo\x00_bar",
			quotePart:   wrapDoubleQuotes,
			expectError: true,
		},
		{
			name:        "too long",
			input:       "a23456789012345678901234567890123456789012345678901234567890123",
			quotePart:   wrapBackticks,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := internal.CleanNamespacedIdentifier(test.input, test.quotePart)
			if test.expectError && err == nil {
				t.Fatal("expected an error but got nil")
			} else if !test.expectError && err != nil {
				t.Fatalf("unexpected error %v", err)
			} else if test.expectError && err != nil && !errors.Is(err, internal.ErrDataInvalid) {
				t.Fatalf("expected error (%v) to match %v", err, internal.ErrDataInvalid)
			}

			if got != test.expected {
				t.Errorf("wrong output; got %q, expected %q", got, test.expected)
			}
		})
	}
}
