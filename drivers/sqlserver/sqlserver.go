// Package sqlserver provides a [godfish.Driver] for sqlserver databases.
package sqlserver

import (
	"database/sql"
	"errors"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

// NewDriver creates a new Microsoft SQL Server driver.
func NewDriver() godfish.Driver { return &driver{} }

// driver implements the godfish.Driver interface for Microsoft SQL Server.
type driver struct {
	connection *sql.DB
}

func (d *driver) Name() string { return "sqlserver" }
func (d *driver) Connect(dsn string) (err error) {
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

func (d *driver) Close() (err error) {
	conn := d.connection
	if conn == nil {
		return
	}
	d.connection = nil
	err = conn.Close()
	return
}

func (d *driver) Execute(query string, args ...any) (err error) {
	_, err = d.connection.Exec(query)
	return
}

func (d *driver) CreateSchemaMigrationsTable(migrationsTable string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	q := `IF OBJECT_ID(@p1, 'U') IS NULL
	CREATE TABLE ` + cleanedTableName + ` (migration_id VARCHAR(128) PRIMARY KEY NOT NULL)`

	_, err = d.connection.Exec(q, cleanedTableName)
	return
}

func (d *driver) AppliedVersions(migrationsTable string) (out godfish.AppliedVersions, err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `SELECT migration_id FROM ` + cleanedTableName + ` ORDER BY migration_id ASC`
	rows, err := d.connection.Query(q)

	var ierr mssql.Error
	// https://docs.microsoft.com/en-us/sql/relational-databases/errors-events/database-engine-events-and-errors
	// Invalid object name '${migrationsTable}'
	if errors.As(err, &ierr) && ierr.SQLErrorNumber() == 208 && strings.Contains(ierr.Error(), migrationsTable) {
		err = godfish.ErrSchemaMigrationsDoesNotExist
	}
	out = godfish.AppliedVersions(rows)
	return
}

func (d *driver) UpdateSchemaMigrations(migrationsTable string, forward bool, version string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	conn := d.connection
	var q string
	if forward {
		// #nosec G202 -- table name was sanitized
		q = `INSERT INTO ` + cleanedTableName + ` (migration_id) VALUES (@p1)`
		_, err = conn.Exec(q, version)
	} else {
		// #nosec G202 -- table name was sanitized
		q = `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = @p1`
		_, err = conn.Exec(q, version)
	}
	return
}

func quotePart(part string) string { return `[` + part + `]` }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
