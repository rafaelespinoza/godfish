// Package ql provides a [driver.Driver] for ql databases.
package ql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/rafaelespinoza/godfish/driver"
	"github.com/rafaelespinoza/godfish/drivers/internal"

	_ "modernc.org/ql/driver" // register driver with database/sql
)

const msgPrefix = "godfish/ql"

// SampleDSN is an example data source name.
const SampleDSN = `file://path/to/db.ql`

// NewDriver creates a new ql driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [driver.Driver] interface for ql databases.
type Driver struct {
	connection *sql.DB
}

func (d *Driver) Name() string { return "ql" }
func (d *Driver) Connect(dsn string) (err error) {
	if d.connection != nil {
		return
	}

	// The ql library has several name options to register with the database/sql
	// package. Going with ql2 because, according to the documentation, this
	// one uses a newer, v2 format, and the driver should be compatible with v1
	// data. The name differs from the value returned by [Driver.Name], which
	// was chosen to match the name of this package.
	conn, err := sql.Open("ql2", dsn)
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
	if err != nil {
		return fmt.Errorf("%s: from Execute: %w", msgPrefix, err)
	}
	return
}

func (d *Driver) CreateSchemaMigrationsTable(ctx context.Context, migrationsTable string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		err = fmt.Errorf("%s: from CreateSchemaMigrationsTable: %w", msgPrefix, err)
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `CREATE TABLE IF NOT EXISTS ` + cleanedTableName + ` (
	migration_id string,
	label string,
	executed_at int64
)`
	_, err = d.connection.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("%s: from CreateSchemaMigrationsTable: %w", msgPrefix, err)
	}
	return
}

func (d *Driver) AppliedVersions(ctx context.Context, migrationsTable string) (out driver.AppliedVersions, err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		err = fmt.Errorf("%s: from AppliedVersions: %w", msgPrefix, err)
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `SELECT migration_id, label, executed_at FROM ` + cleanedTableName + ` ORDER BY migration_id`
	rows, err := d.connection.QueryContext(ctx, q)
	if err != nil {
		m := err.Error()
		if strings.Contains(m, "does not exist") {
			return nil, fmt.Errorf("%s: from AppliedVersions %w %w", msgPrefix, err, driver.ErrSchemaMigrationsDoesNotExist)
		}

		return nil, fmt.Errorf("%s: from AppliedVersions: %w", msgPrefix, err)

	}
	out = driver.AppliedVersions(rows)
	return
}

func (d *Driver) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version, label string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		err = fmt.Errorf("%s: from UpdateSchemaMigrations %w", msgPrefix, err)
		return
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("%s: UpdateSchemaMigrations (fwd?=%t): %w", msgPrefix, forward, err)
		}
	}()

	conn := d.connection
	if !forward {
		// #nosec G202 -- table name was sanitized
		q := `DELETE FROM ` + cleanedTableName + ` WHERE migration_id == $1`
		_, err = conn.ExecContext(ctx, q, version)
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `INSERT INTO ` + cleanedTableName + ` (migration_id, label, executed_at) VALUES ($1, $2, $3)`
	now := time.Now().UTC()
	_, err = conn.ExecContext(ctx, q, version, label, now.Unix())
	return
}

// UpgradeSchemaMigrations is a no op. The ql driver was introduced after
// v0.14.0, so it is not expected to be on the legacy schema.
func (d *Driver) UpgradeSchemaMigrations(ctx context.Context, migrationsTable string) error {
	slog.Info(fmt.Sprintf("%s: from UpgradeSchemaMigrations, this DB driver was added after v0.14.0; no op", msgPrefix))
	return nil
}

// In ql, guillmets («») denote quoted identifiers.
// > A quoted identifier is a string of any charaters between guillmets «»
// source: https://pkg.go.dev/modernc.org/ql#hdr-Identifiers
const quoteCharBegin, quoteCharEnd = `«`, `»`

func quotePart(part string) string { return quoteCharBegin + part + quoteCharEnd }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
