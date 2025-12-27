// Package sqlite3 provides a [godfish.Driver] for sqlite3 databases.
package sqlite3

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/rafaelespinoza/godfish"
	sqlib "modernc.org/sqlite"
)

// NewDriver creates a new sqlite3 driver.
func NewDriver() godfish.Driver { return &driver{} }

// driver implements the Driver interface for sqlite3 databases.
type driver struct {
	connection *sql.DB
}

func (d *driver) Name() string { return "sqlite" }
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

	var ierr *sqlib.Error
	if errors.As(err, &ierr) && ierr.Code() == 1 && strings.Contains(ierr.Error(), "no such table") {
		err = godfish.ErrSchemaMigrationsDoesNotExist
	}

	out = godfish.AppliedVersions(rows)
	return
}

func (d *driver) UpdateSchemaMigrations(migrationsTable string, forward bool, version string) (err error) {
	conn := d.connection
	if forward {
		_, err = conn.Exec(
			`INSERT INTO `+migrationsTable+` (migration_id) VALUES ($1)`,
			version,
		)
	} else {
		_, err = conn.Exec(
			`DELETE FROM `+migrationsTable+` WHERE migration_id = $1`,
			version,
		)
	}
	return
}
