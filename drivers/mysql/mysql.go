// Package mysql provides a [godfish.Driver] for mysql-compatible databases.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"

	_ "github.com/go-sql-driver/mysql" // register driver with database/sql
)

const msgPrefix = "mysql: "

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

	metadata, err := checkSchemaMigrationMetadata(ctx, d, cleanedTableName)
	if err != nil {
		return
	} else if !metadata.hasTable {
		err = godfish.ErrSchemaMigrationsDoesNotExist
		return
	} else if !metadata.hasColLabel || !metadata.hasColExecutedAt {
		err = godfish.ErrSchemaMigrationsMissingColumns
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `SELECT migration_id, label, executed_at FROM ` + cleanedTableName + ` ORDER BY migration_id ASC`
	rows, err := d.connection.QueryContext(ctx, q)
	out = godfish.AppliedVersions(rows)
	return
}

func (d *Driver) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version, label string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	conn := d.connection
	if !forward {
		// #nosec G202 -- table name was sanitized
		q := `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = ?`
		_, err = conn.ExecContext(ctx, q, version)
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `INSERT INTO ` + cleanedTableName + ` (migration_id, label, executed_at) VALUES (?, ?, ?)`
	now := time.Now().UTC()
	_, err = conn.ExecContext(ctx, q, version, label, now.Unix())
	return
}

func (d *Driver) UpgradeSchemaMigrations(ctx context.Context, migrationsTable string) error {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return err
	}
	const errMsgPrefix = msgPrefix + "upgrading schema migrations table"

	// #nosec G202 -- table name was sanitized
	q := `ALTER TABLE ` + cleanedTableName + `
	ADD COLUMN label VARCHAR(255) DEFAULT '',
	ADD COLUMN executed_at BIGINT DEFAULT 0`

	if _, err = d.connection.ExecContext(ctx, q); err != nil {
		err = fmt.Errorf(errMsgPrefix+", exec failed; %w", err)
	}

	return err
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
		msgPrefix+"checking for table, column existence",
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
