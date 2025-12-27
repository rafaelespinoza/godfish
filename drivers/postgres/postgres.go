// Package postgres provides a [godfish.Driver] for postgres databases.
package postgres

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/rafaelespinoza/godfish"
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
	_, err = d.connection.Exec(
		`CREATE TABLE IF NOT EXISTS ` + migrationsTable + ` (
			migration_id VARCHAR(128) PRIMARY KEY NOT NULL
		)`)
	return
}

func (d *driver) AppliedVersions(migrationsTable string) (out godfish.AppliedVersions, err error) {
	rows, err := d.connection.Query(
		`SELECT migration_id FROM ` + migrationsTable + ` ORDER BY migration_id ASC`,
	)
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
	conn := d.connection
	if forward {
		_, err = conn.Exec(`
			INSERT INTO `+migrationsTable+` (migration_id)
			VALUES ($1)
			RETURNING migration_id`,
			version,
		)
	} else {
		_, err = conn.Exec(`
			DELETE FROM `+migrationsTable+`
			WHERE migration_id = $1
			RETURNING migration_id`,
			version,
		)
	}
	return
}
