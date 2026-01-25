// Package postgres provides a [godfish.Driver] for postgres databases.
package postgres

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

// NewDriver creates a new postgres driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [godfish.Driver] interface for postgres databases.
type Driver struct {
	connection *sql.DB
}

func (d *Driver) Name() string { return "postgres" }
func (d *Driver) Connect(dsn string) (err error) {
	if d.connection != nil {
		return
	}
	conn, err := sql.Open(d.Name(), dsn)
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

	q := `CREATE TABLE IF NOT EXISTS ` + cleanedTableName + ` (migration_id VARCHAR(128) PRIMARY KEY NOT NULL)`
	_, err = d.connection.ExecContext(ctx, q)
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
	if ierr, ok := err.(*pq.Error); ok {
		// https://www.postgresql.org/docs/current/errcodes-appendix.html
		if ierr.Code == "42P01" {
			err = godfish.ErrSchemaMigrationsDoesNotExist
		}
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
		q = `INSERT INTO ` + cleanedTableName + ` (migration_id) VALUES ($1) RETURNING migration_id`
		_, err = conn.ExecContext(ctx, q, version)
	} else {
		// #nosec G202 -- table name was sanitized
		q = `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = $1 RETURNING migration_id`
		_, err = conn.ExecContext(ctx, q, version)
	}
	return
}

func quotePart(part string) string { return pq.QuoteIdentifier(part) }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
