package mysql

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	my "github.com/go-sql-driver/mysql"
	"github.com/rafaelespinoza/godfish"
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

func (d *driver) Execute(query string, args ...interface{}) (err error) {
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

func (d *driver) CreateSchemaMigrationsTable() (err error) {
	_, err = d.connection.Exec(
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_id VARCHAR(128) PRIMARY KEY NOT NULL
		)`)
	return
}

func (d *driver) AppliedVersions() (out godfish.AppliedVersions, err error) {
	rows, err := d.connection.Query(
		`SELECT migration_id FROM schema_migrations ORDER BY migration_id ASC`,
	)
	if ierr, ok := err.(*my.MySQLError); ok {
		// https://dev.mysql.com/doc/refman/8.0/en/server-error-reference.html#error_er_no_such_table
		if ierr.Number == 1146 {
			err = godfish.ErrSchemaMigrationsDoesNotExist
		}
	}
	out = godfish.AppliedVersions(rows)
	return
}

func (d *driver) UpdateSchemaMigrations(forward bool, version string) (err error) {
	conn := d.connection
	if forward {
		_, err = conn.Exec(`
			INSERT INTO schema_migrations (migration_id)
			VALUES (?)`,
			version,
		)
	} else {
		_, err = conn.Exec(`
			DELETE FROM schema_migrations
			WHERE migration_id = ?`,
			version,
		)
	}
	return
}
