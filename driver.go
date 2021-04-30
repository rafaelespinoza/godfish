package godfish

// Driver adapts a database implementation to use godfish.
type Driver interface {
	// Name should return the name of the driver: ie: postgres, mysql, etc
	Name() string

	// Connect should open a connection to the database.
	Connect(dsn string) error
	// Close should close the database connection.
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
