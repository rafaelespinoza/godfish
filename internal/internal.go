// Package internal defines common functionality available within the library.
package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// Config is for various runtime settings.
type Config struct {
	PathToFiles     string `json:"path_to_files"`
	ForwardLabel    string `json:"forward_label"`
	ReverseLabel    string `json:"reverse_label"`
	MigrationsTable string `json:"migrations_table"`
}

// LogValue lets this type implement the [slog.LogValuer] interface.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("path_to_files", c.PathToFiles),
		slog.String("forward_label", c.ForwardLabel),
		slog.String("reverse_label", c.ReverseLabel),
		slog.String("migrations_table", c.MigrationsTable),
	)
}

// General error values to help shape behavior.
var (
	ErrNotFound           = errors.New("not found")
	ErrDataInvalid        = errors.New("data invalid")
	ErrExecutingMigration = errors.New("executing migration")
)

const (
	// DSNKey is the name of the environment variable for connecting to the DB.
	DSNKey = "DB_DSN"

	// DefaultMigrationsTableName is the default name of the database table for
	// storing database migration state.
	DefaultMigrationsTableName = "schema_migrations"
)

// identifierMatcher meant to as a conservative approach in matching names for
// any DB driver that might use the library. The first character must be an
// ASCII letter, while the remaining must be alphanumeric. The max length for an
// identifier in most DBs is 64, but is actually 63 (bytes) in a default
// Postgres configuration (see documentation for `max_identifier_length`).
// The length allowed here is even shorter than that for reasons.
var identifierMatcher = regexp.MustCompile(`^[a-z][a-z0-9_]{0,61}$`)

// CleanNamespacedIdentifier sanitizes and formats a potentially namespaced
// DB identifier. It is meant for implementations of [godfish.Driver].
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
		return "", fmt.Errorf("%w, identifier can only have 1 or 2 parts separated by %q", ErrDataInvalid, ".")
	}

	cleanedParts := make([]string, len(parts))
	for i, part := range parts {
		for _, ch := range []string{"`", `"`, "[", "]"} {
			part = strings.ReplaceAll(part, ch, "")
		}
		part = strings.ToLower(strings.TrimSpace(part))

		if !identifierMatcher.MatchString(part) {
			return "", fmt.Errorf("%w: identifier part must match pattern %s", ErrDataInvalid, identifierMatcher.String())
		}
		cleanedParts[i] = quotePart(part)
	}

	return strings.Join(cleanedParts, "."), nil
}
