// Package sqlite3 provides a [godfish.Driver] for sqlite3 databases.
package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"

	_ "modernc.org/sqlite" // register driver with database/sql
)

const msgPrefix = "sqlite3: "

// NewDriver creates a new sqlite3 driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [godfish.Driver] interface for sqlite3 databases.
type Driver struct {
	connection *sql.DB
}

func (d *Driver) Name() string { return "sqlite" }
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
		q := `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = $1`
		_, err = conn.ExecContext(ctx, q, version)
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `INSERT INTO ` + cleanedTableName + ` (migration_id, label, executed_at) VALUES ($1, $2, $3)`
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

	tx, terr := d.connection.BeginTx(ctx, nil)
	if terr != nil {
		return fmt.Errorf(errMsgPrefix+", beginning transaction; %w", terr)
	}

	// sqlite3 can do transaction DDL, but each column must be added in its own query.

	querySuffixes := []string{
		`ADD COLUMN label VARCHAR(255) DEFAULT ''`,
		`ADD COLUMN executed_at BIGINT DEFAULT 0`,
	}
	for _, qs := range querySuffixes {
		// #nosec G202 -- table name was sanitized
		q := `ALTER TABLE ` + cleanedTableName + ` ` + qs
		_, xerr := tx.ExecContext(ctx, q)
		if xerr != nil {
			if rerr := tx.Rollback(); rerr != nil {
				return fmt.Errorf(errMsgPrefix+", exec and rollback failed, exec error (%w), rollback error (%w) ", xerr, rerr)
			}
			return fmt.Errorf(errMsgPrefix+", exec failed but fortunately the rollback was OK; exec error %w", xerr)
		}
	}

	cerr := tx.Commit()
	if cerr != nil {
		cerr = fmt.Errorf(errMsgPrefix+", during commit; %w", cerr)
	}
	return cerr
}

type metadataResult struct {
	hasTable         bool
	hasColLabel      bool
	hasColExecutedAt bool
}

// checkSchemaMigrationMetadata inspects the shape of tableName to see if it has two
// columns: label and executed_at. These results inform the tool about the need
// to upgrade the schema migrations table.
func checkSchemaMigrationMetadata(ctx context.Context, d *Driver, tableName string) (out metadataResult, err error) {
	// Expect for the input tableName to have been treated by cleanIdentifier.
	// It doesn't need to be quoted in this case because it's used as a regular
	// query parameter in this function.
	tableName = strings.ReplaceAll(tableName, quote, "")

	lgr := slog.With("driver", d.Name(), slog.String("table_name", tableName))

	const query = `
SELECT m.name AS table_name, p.name AS column_name
FROM sqlite_master m LEFT JOIN pragma_table_info(m.name) p
    ON p.name IN (?, ?)
WHERE m.type = 'table'
  AND m.name = ?`
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

const quote = `"`

func quotePart(part string) string { return quote + part + quote }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
