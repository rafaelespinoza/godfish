// Package sqlserver provides a [godfish.Driver] for sqlserver databases.
package sqlserver

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

// NewDriver creates a new Microsoft SQL Server driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [godfish.Driver] interface for Microsoft SQL Server.
type Driver struct {
	connection *sql.DB
}

func (d *Driver) Name() string { return "sqlserver" }
func (d *Driver) Connect(dsn string) (err error) {
	if d.connection != nil {
		return
	}

	// github.com/denisenkom/go-mssqldb registers two sql.Driver names:
	// "mssql" and "sqlserver". Their docs seem to steer users towards the
	// latter, so just use that one.
	conn, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return
	}
	d.connection = conn
	return
}

func (d *Driver) Close() (err error) {
	conn := d.connection
	if conn == nil {
		return
	}
	d.connection = nil
	err = conn.Close()
	return
}

func (d *Driver) Execute(ctx context.Context, query string, args ...any) (err error) {
	_, err = d.connection.ExecContext(ctx, query)
	return
}

func (d *Driver) CreateSchemaMigrationsTable(ctx context.Context, migrationsTable string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	q := `IF OBJECT_ID(@p1, 'U') IS NULL
	CREATE TABLE ` + cleanedTableName + ` (migration_id VARCHAR(128) PRIMARY KEY NOT NULL)`

	_, err = d.connection.ExecContext(ctx, q, cleanedTableName)
	return
}

func (d *Driver) AppliedVersions(ctx context.Context, migrationsTable string) (out godfish.AppliedVersions, err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `SELECT migration_id FROM ` + cleanedTableName + ` ORDER BY migration_id ASC`
	rows, err := d.connection.QueryContext(ctx, q)

	var ierr mssql.Error
	// https://docs.microsoft.com/en-us/sql/relational-databases/errors-events/database-engine-events-and-errors
	// Invalid object name '${migrationsTable}'
	if errors.As(err, &ierr) && ierr.SQLErrorNumber() == 208 && strings.Contains(ierr.Error(), migrationsTable) {
		err = godfish.ErrSchemaMigrationsDoesNotExist
	}
	out = godfish.AppliedVersions(rows)
	return
}

func (d *Driver) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	conn := d.connection
	var q string
	if forward {
		// #nosec G202 -- table name was sanitized
		q = `INSERT INTO ` + cleanedTableName + ` (migration_id) VALUES (@p1)`
		_, err = conn.ExecContext(ctx, q, version)
	} else {
		// #nosec G202 -- table name was sanitized
		q = `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = @p1`
		_, err = conn.ExecContext(ctx, q, version)
	}
	return
}

func quotePart(part string) string { return `[` + part + `]` }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
