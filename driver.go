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
	Connect() (*sql.DB, error)
	// Close should check if there's an internal reference to a database
	// connection (a *sql.DB) and if it's present, close it. Then reset the
	// internal reference to that connection to nil.
	Close() error
	// DSN returns data source name info, ie: how do I connect?
	DSN() DSN

	// AppliedVersions queries the schema migrations table for migration
	// versions that have been executed against the database. If the schema
	// migrations table does not exist, the returned error should be
	// ErrSchemaMigrationsDoesNotExist.
	AppliedVersions() (AppliedVersions, error)
	// CreateSchemaMigrationsTable should create a table to record migration
	// versions once they've been applied. The version should be a timestamp as
	// a string, formatted as the TimeFormat variable in this package.
	CreateSchemaMigrationsTable() error
	// DumpSchema should output the database structure to stdout.
	DumpSchema() error
	// Execute runs the schema change and commits it to the database. The query
	// parameter is a SQL string and may contain placeholders for the values in
	// args. Input should be passed to conn so it could be sanitized, escaped.
	Execute(query string, args ...interface{}) error
	// UpdateSchemaMigrations records a timestamped version of a migration that
	// has been successfully applied by adding a new row to the schema
	// migrations table.
	UpdateSchemaMigrations(dir Direction, version string) error
}

// NewDriver initializes a Driver implementation by name and connection
// parameters.
func NewDriver(dsn DSN, migConf *MigrationsConf) (driver Driver, err error) {
	return dsn.NewDriver(migConf)
}

// DSN generates a data source name or connection URL for DB connections. The
// output will be passed to the standard library's sql.Open method.
type DSN interface {
	// Boot takes inputs from the host environment so it can create a Driver.
	//
	// Deprecated: Set the DB_DSN environment variable instead of using this.
	Boot(ConnectionParams) error
	// NewDriver calls the constructor of the corresponding Driver.
	NewDriver(*MigrationsConf) (Driver, error)
	// String uses connection parameters to form the data source name.
	String() string
}

// ConnectionParams is what to use when initializing a DSN.
//
// Deprecated: Set the DB_DSN environment variable instead of using this.
type ConnectionParams struct {
	Encoding string // Encoding is the client encoding for the connection.
	Host     string // Host is the name of the host to connect to.
	Name     string // Name is the database name.
	Pass     string // Pass is the password to use for the connection.
	Port     string // Port is the connection port.
	User     string // User is the name of the user to connect as.
}

// MigrationsConf is intended to lend customizations such as specifying the path
// to the migration files.
type MigrationsConf struct {
	PathToFiles string `json:"path_to_files"`
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
