// Package mysql provides a [godfish.Driver] for mysql-compatible databases.
package mysql

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	my "github.com/go-sql-driver/mysql"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

// NewDriver creates a new mysql driver.
func NewDriver() godfish.Driver { return &driver{} }

// driver implements the godfish.Driver interface for mysql databases.
type driver struct {
	connection *sql.DB
}

func (d *driver) Name() string { return "mysql" }
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

var statementDelimiter = regexp.MustCompile(`;\s*\n`)

func (d *driver) Execute(query string, args ...any) (err error) {
	// Attempt to support migrations with 1 or more statements. AFAIK, the
	// standard library does not support executing multiple statements at once.
	// As a workaround, break them up and apply them.
	statements := statementDelimiter.Split(query, -1)
	if len(statements) < 1 {
		return
	}
	tx, err := d.connection.Begin()
	if err != nil {
		return
	}
	for _, q := range statements {
		if len(strings.TrimSpace(q)) < 1 {
			continue
		}
		_, err = tx.Exec(q)
		if err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				return fmt.Errorf("%w; %v", err, rerr)
			}
			return
		}
	}
	return tx.Commit()
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
	if ierr, ok := err.(*my.MySQLError); ok {
		// https://dev.mysql.com/doc/refman/8.0/en/server-error-reference.html#error_er_no_such_table
		if ierr.Number == 1146 {
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
		q = `INSERT INTO ` + cleanedTableName + ` (migration_id) VALUES (?)`
		_, err = conn.Exec(q, version)
	} else {
		// #nosec G202 -- table name was sanitized
		q = `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = ?`
		_, err = conn.Exec(q, version)
	}
	return
}

func quotePart(part string) string { return "`" + part + "`" }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
