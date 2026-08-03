// Package mysql provides a [driver.Driver] for mysql-compatible databases.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/rafaelespinoza/godfish/driver"
	"github.com/rafaelespinoza/godfish/drivers/internal"

	_ "github.com/go-sql-driver/mysql" // register driver with database/sql
)

const (
	// msgPrefix denotes the origin of a log or error.
	msgPrefix = "godfish/mysql"

	// driverMsgPrefix is for logs or errors emitted from a Driver method.
	driverMsgPrefix = msgPrefix + ".Driver"
)

// SampleDSN is an example data source name.
const SampleDSN = `username:password@tcp(server_host)/db_name?param1=value&paramN=valueN` // #nosec G101 -- not real credentials.

// NewDriver creates a new mysql driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [driver.Driver] interface for mysql databases.
type Driver struct {
	connection *sql.DB
}

func (d *Driver) Name() string { return "mysql" }
func (d *Driver) Connect(dsn string) error {
	if d.connection != nil {
		return nil
	}
	conn, err := sql.Open(d.Name(), dsn)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "Connect", err)
	}
	d.connection = conn
	return nil
}

func (d *Driver) Close() (err error) {
	conn := d.connection
	if conn == nil {
		return
	}
	d.connection = nil
	if err = conn.Close(); err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "Close", err)
	}
	return nil
}

var statementDelimiter = regexp.MustCompile(`;\s*\n`)

func (d *Driver) Execute(ctx context.Context, query string, args ...any) error {
	// Attempt to support migrations with 1 or more statements. AFAIK, the
	// standard library does not support executing multiple statements at once.
	// As a workaround, break them up and apply them.
	statements := statementDelimiter.Split(query, -1)
	if len(statements) < 1 {
		return nil
	}
	tx, err := d.connection.Begin()
	if err != nil {
		return fmt.Errorf("%s.%s: beginning tx: %w", driverMsgPrefix, "Execute", err)
	}
	for _, q := range statements {
		if len(strings.TrimSpace(q)) < 1 {
			continue
		}
		_, err = tx.ExecContext(ctx, q)
		if err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				return fmt.Errorf("%s.%s: rolling back tx: original error [%w], rollback error [%w]", driverMsgPrefix, "Execute", err, rerr)
			}
			return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "Execute", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%s.%s: committing tx: %w", driverMsgPrefix, "Execute", err)
	}
	return nil
}

func (d *Driver) CreateSchemaMigrationsTable(ctx context.Context, migrationsTable string) error {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "CreateSchemaMigrationsTable", err)
	}

	// #nosec G202 -- table name was sanitized
	q := `CREATE TABLE IF NOT EXISTS ` + cleanedTableName + ` (
	migration_id VARCHAR(128) PRIMARY KEY NOT NULL,
	label VARCHAR(255) DEFAULT '',
	executed_at BIGINT DEFAULT 0
)`
	if _, err = d.connection.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "CreateSchemaMigrationsTable", err)
	}
	return nil
}

func (d *Driver) AppliedVersions(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: %w", driverMsgPrefix, "AppliedVersions", err)
	}

	metadata, err := checkSchemaMigrationMetadata(ctx, d, cleanedTableName)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: checking schema migration metadata: %w", driverMsgPrefix, "AppliedVersions", err)
	} else if !metadata.hasTable {
		err = fmt.Errorf("%s.%s: %w", driverMsgPrefix, "AppliedVersions", driver.ErrSchemaMigrationsDoesNotExist)
		return nil, err
	} else if !metadata.hasColLabel || !metadata.hasColExecutedAt {
		err = fmt.Errorf("%s.%s: %w", driverMsgPrefix, "AppliedVersions", driver.ErrSchemaMigrationsMissingColumns)
		return nil, err
	}

	// #nosec G202 -- table name was sanitized
	q := `SELECT migration_id, label, executed_at FROM ` + cleanedTableName + ` ORDER BY migration_id ASC`
	rows, err := d.connection.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: querying: %w", driverMsgPrefix, "AppliedVersions", err)
	}

	return driver.AppliedVersions(rows), nil
}

func (d *Driver) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "UpdateSchemaMigrations", err)
	}

	conn := d.connection
	if !forward {
		// #nosec G202 -- table name was sanitized
		q := `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = ?`
		if _, err = conn.ExecContext(ctx, q, version); err != nil {
			return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "UpdateSchemaMigrations", err)
		}
		return nil
	}

	// #nosec G202 -- table name was sanitized
	q := `INSERT INTO ` + cleanedTableName + ` (migration_id, label, executed_at) VALUES (?, ?, ?)`
	now := time.Now().UTC()
	if _, err = conn.ExecContext(ctx, q, version, label, now.Unix()); err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "UpdateSchemaMigrations", err)
	}
	return nil
}

func (d *Driver) UpgradeSchemaMigrations(ctx context.Context, migrationsTable string) error {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "UpgradeSchemaMigrations", err)
	}
	const errMsgPrefix = driverMsgPrefix + ".UpgradeSchemaMigrations"

	// #nosec G202 -- table name was sanitized
	q := `ALTER TABLE ` + cleanedTableName + `
	ADD COLUMN label VARCHAR(255) DEFAULT '',
	ADD COLUMN executed_at BIGINT DEFAULT 0`

	if _, err = d.connection.ExecContext(ctx, q); err != nil {
		return fmt.Errorf(errMsgPrefix+": exec failed; %w", err)
	}

	return nil
}

type metadataResult struct {
	hasTable         bool
	hasColLabel      bool
	hasColExecutedAt bool
}

func checkSchemaMigrationMetadata(ctx context.Context, d *Driver, tableName string) (out metadataResult, err error) {
	// Expect for the input tableName to have been treated by cleanIdentifier.
	// It doesn't need to be quoted in this case because it's used as a regular
	// query parameter in this function.
	tableName = strings.ReplaceAll(tableName, quote, "")

	lgr := slog.With("driver", d.Name(), slog.String("table_name", tableName))

	const query = `
SELECT t.table_name, c.column_name
FROM information_schema.tables t LEFT JOIN information_schema.columns c
	ON  t.table_schema = c.table_schema
	AND t.table_name = c.table_name
	AND c.column_name IN (?, ?)
WHERE t.table_schema = DATABASE()
	AND t.table_name = ?
`
	args := []any{"label", "executed_at", tableName}
	lgr.Debug(
		"checking for table, column existence",
		slog.String("query", query), slog.Any("args", args),
	)
	rows, err := d.connection.QueryContext(ctx, query, args...)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var table, column sql.NullString
		if err = rows.Scan(&table, &column); err != nil {
			return
		}

		out.hasTable = table.Valid

		if column.Valid {
			switch column.String {
			case "label":
				out.hasColLabel = true
			case "executed_at":
				out.hasColExecutedAt = true
			}
		}
	}

	return
}

const quote = "`"

func quotePart(part string) string { return quote + part + quote }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
