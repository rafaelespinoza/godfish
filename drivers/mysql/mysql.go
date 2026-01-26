// Package mysql provides a [godfish.Driver] for mysql-compatible databases.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	my "github.com/go-sql-driver/mysql"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

// NewDriver creates a new mysql driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [godfish.Driver] interface for mysql databases.
type Driver struct {
	connection *sql.DB
}

func (d *Driver) Name() string { return "mysql" }
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

var statementDelimiter = regexp.MustCompile(`;\s*\n`)

func (d *Driver) Execute(ctx context.Context, query string, args ...any) (err error) {
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
		_, err = tx.ExecContext(ctx, q)
		if err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				return fmt.Errorf("%w; %v", err, rerr)
			}
			return
		}
	}
	return tx.Commit()
}

func (d *Driver) CreateSchemaMigrationsTable(ctx context.Context, migrationsTable string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	q := `CREATE TABLE IF NOT EXISTS ` + cleanedTableName + ` (
	migration_id VARCHAR(128) PRIMARY KEY NOT NULL,
	label VARCHAR(255) DEFAULT '',
	executed_at BIGINT DEFAULT 0
)`
	_, err = d.connection.ExecContext(ctx, q)
	return
}

func (d *Driver) AppliedVersions(ctx context.Context, migrationsTable string) (out godfish.AppliedVersions, err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `SELECT migration_id, label, executed_at FROM ` + cleanedTableName + ` ORDER BY migration_id ASC`
	rows, err := d.connection.QueryContext(ctx, q)
	if ierr, ok := err.(*my.MySQLError); ok {
		// https://dev.mysql.com/doc/refman/8.0/en/server-error-reference.html#error_er_no_such_table
		if ierr.Number == 1146 {
			err = godfish.ErrSchemaMigrationsDoesNotExist
		}
	}
	out = godfish.AppliedVersions(rows)
	return
}

func (d *Driver) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version, label string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	conn := d.connection
	var q string
	if forward {
		// #nosec G202 -- table name was sanitized
		q = `INSERT INTO ` + cleanedTableName + ` (migration_id, label, executed_at) VALUES (?, ?, ?)`
		now := time.Now().UTC()
		_, err = conn.ExecContext(ctx, q, version, label, now.Unix())
	} else {
		// #nosec G202 -- table name was sanitized
		q = `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = ?`
		_, err = conn.ExecContext(ctx, q, version)
	}
	return
}

func quotePart(part string) string { return "`" + part + "`" }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
