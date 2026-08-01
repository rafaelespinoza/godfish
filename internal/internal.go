// Package internal defines common functionality available within the library.
package internal

import (
	"errors"
	"log/slog"
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

// IsInvalidDataError checks if err is an [ErrDataInvalid], and if not then it
// checks if the errors implements the interface:
//
//	Invalid() bool
//
// and if it does not, then it checks if the error wraps an error implemnting
// that interface. It's made for the core library to recognize error signals
// returned by Drivers, without necessarily requiring them to know about one
// another.
func IsInvalidDataError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrDataInvalid) {
		return true
	}

	// For implicit compatibility with drivers that may return certain errors
	// targeted by the core library, check for this behavior.
	type invalidErr interface {
		Invalid() bool
	}

	// Does this error itself implement the interface?
	terr, ok := err.(invalidErr)
	if ok && terr.Invalid() {
		return true
	}

	// Does this error wrap another error that implements the interface?
	uerr := errors.Unwrap(err)
	verr, ok := uerr.(invalidErr)
	return ok && verr.Invalid()
}

const (
	// DSNKey is the name of the environment variable for connecting to the DB.
	DSNKey = "DB_DSN"

	// DefaultMigrationsTableName is the default name of the database table for
	// storing database migration state.
	DefaultMigrationsTableName = "schema_migrations"
)
