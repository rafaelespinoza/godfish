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
	ErrExecutingMigration = errors.New("executing migration")
	ErrDataInvalid        = errors.New("data invalid")
)

// IsInvalidDataError checks if err is an [ErrDataInvalid], and if not then it
// checks if the error or a wrapped error implements the interface:
//
//	Invalid() bool
//
// It's made for the core library to recognize error signals returned by
// Drivers, without necessarily requiring them to know about one another.
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

	var ierr invalidErr
	return errors.As(err, &ierr) && ierr.Invalid()
}

const (
	// DSNKey is the name of the environment variable for connecting to the DB.
	DSNKey = "DB_DSN"

	// DefaultMigrationsTableName is the default name of the database table for
	// storing database migration state.
	DefaultMigrationsTableName = "schema_migrations"
)
