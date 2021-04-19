package godfish

import "database/sql"

// A Driver describes what a database driver (anything at
// https://github.com/golang/go/wiki/SQLDrivers) should be able to do.
type Driver interface {
	// Name should return the name of the driver: ie: postgres, mysql, etc
	Name() string

	// Connect should open a connection (a *sql.DB) to the database and save an
	// internal reference to that connection for later use. This library might
	// call this method multiple times, so use the internal reference if it's
	// present instead of reconnecting to the database.
	Connect(dsn string) (*sql.DB, error)
	// Close should check if there's an internal reference to a database
	// connection (a *sql.DB) and if it's present, close it. Then reset the
	// internal reference to that connection to nil.
	Close() error

	// AppliedVersions queries the schema migrations table for migration
	// versions that have been executed against the database. If the schema
	// migrations table does not exist, the returned error should be
	// ErrSchemaMigrationsDoesNotExist.
	AppliedVersions() (AppliedVersions, error)
	// CreateSchemaMigrationsTable should create a table to record migration
	// versions once they've been applied. The version should be a timestamp as
	// a string, formatted as the TimeFormat variable in this package.
	CreateSchemaMigrationsTable() error
	// Execute runs the schema change and commits it to the database. The query
	// parameter is a SQL string and may contain placeholders for the values in
	// args. Input should be passed to conn so it could be sanitized, escaped.
	Execute(query string, args ...interface{}) error
	// UpdateSchemaMigrations records a timestamped version of a migration that
	// has been successfully applied by adding a new row to the schema
	// migrations table.
	UpdateSchemaMigrations(dir Direction, version string) error
}

// AppliedVersions represents an iterative list of migrations that have been run
// against the database and have been recorded in the schema migrations table.
// It's enough to convert a *sql.Rows struct when implementing the Driver
// interface since a *sql.Rows already satisfies this interface. See existing
// Driver implementations in this package for examples.
type AppliedVersions interface {
	Close() error
	Next() bool
	Scan(dest ...interface{}) error
}

var _ AppliedVersions = (*sql.Rows)(nil)
