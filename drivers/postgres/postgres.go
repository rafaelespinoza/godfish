// Package postgres provides a [godfish.Driver] for postgres databases.
package postgres

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

// NewDriver creates a new postgres driver.
func NewDriver() godfish.Driver { return &driver{} }

// driver implements the Driver interface for postgres databases.
type driver struct {
	connection *sql.DB
}

func (d *driver) Name() string { return "postgres" }
func (d *driver) Connect(dsn string) (err error) {
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

	q := `CREATE TABLE IF NOT EXISTS ` + cleanedTableName + ` (migration_id VARCHAR(128) PRIMARY KEY NOT NULL)`
	_, err = d.connection.Exec(q)
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
	if ierr, ok := err.(*pq.Error); ok {
		// https://www.postgresql.org/docs/current/errcodes-appendix.html
		if ierr.Code == "42P01" {
			err = godfish.ErrSchemaMigrationsDoesNotExist
		}
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
		q = `INSERT INTO ` + cleanedTableName + ` (migration_id) VALUES ($1) RETURNING migration_id`
		_, err = conn.Exec(q, version)
	} else {
		// #nosec G202 -- table name was sanitized
		q = `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = $1 RETURNING migration_id`
		_, err = conn.Exec(q, version)
	}
	return
}

func quotePart(part string) string { return pq.QuoteIdentifier(part) }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
