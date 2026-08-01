// Package internal is helper utilities for supported implementations of the
// godfish Driver interface.
package internal

import (
	"fmt"
	"regexp"
	"strings"
)

// identifierMatcher meant to as a conservative approach in matching names for
// any DB driver that might use the library. The first character must be an
// ASCII letter, while the remaining must be alphanumeric. The max length for an
// identifier in most DBs is 64, but is actually 63 (bytes) in a default
// Postgres configuration (see documentation for `max_identifier_length`).
// The length allowed here is even shorter than that for reasons.
var identifierMatcher = regexp.MustCompile(`^[a-z][a-z0-9_]{0,61}$`)

// CleanNamespacedIdentifier sanitizes and formats a potentially namespaced
// DB identifier. It is meant for [Driver] implementations that want to accept
// user input for identifiers like table names.
//
// It splits the input by the first dot ('.') and validates each part against
// a strict alphanumeric regex pattern (^[a-z][a-z0-9_]{0,61}$). This is
// intended to prevent SQL injection by ensuring no special characters, spaces,
// or comments are present.
//
// After validation, each part is passed to the wrapper function, quotePart,
// to be enclosed in the appropriate database-specific quote character (e.g.,
// backticks for MySQL, double quotes for PostgreSQL/Cassandra, or straight
// brackets for SQL Server).
//
// Constraints:
//
//   - The input may contain at most one dot (e.g., "table" or "schema.table").
//   - Each part must start with a letter and be between 1 and 62 characters long.
//   - Existing quote characters (`, ", [, ]) are stripped before validation
//     to prevent double-quoting or escape attempts.
//
// It returns the fully qualified, quoted identifier or an error if the input
// fails validation.
func CleanNamespacedIdentifier(input string, quotePart func(string) string) (string, error) {
	input = strings.TrimSpace(input)
	parts := strings.Split(input, ".")

	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w, identifier can only have 1 or 2 parts separated by %q", invalidDataError{}, ".")
	}

	cleanedParts := make([]string, len(parts))
	for i, part := range parts {
		for _, ch := range []string{"`", `"`, "[", "]"} {
			part = strings.ReplaceAll(part, ch, "")
		}
		part = strings.ToLower(strings.TrimSpace(part))

		if !identifierMatcher.MatchString(part) {
			return "", fmt.Errorf("%w: identifier part must match pattern %s", invalidDataError{}, identifierMatcher.String())
		}
		cleanedParts[i] = quotePart(part)
	}

	return strings.Join(cleanedParts, "."), nil
}

type invalidDataError struct{}

func (e invalidDataError) Error() string { return "data invalid" }

func (e invalidDataError) Invalid() bool { return true }
