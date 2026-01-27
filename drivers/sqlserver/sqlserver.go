// Package sqlserver provides a [godfish.Driver] for sqlserver databases.
package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"

	_ "github.com/microsoft/go-mssqldb" // register driver with database/sql
)

const msgPrefix = "sqlserver: "

// NewDriver creates a new Microsoft SQL Server driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [godfish.Driver] interface for Microsoft SQL Server.
type Driver struct {
	connection *sql.DB
}

func (d *Driver) Name() string { return "sqlserver" }
func (d *Driver) Connect(dsn string) (err error) {
	if d.connection != nil {
		return
	}

	// github.com/microsoft/go-mssqldb registers two sql.Driver names:
	// "mssql" and "sqlserver".
	// * Using "mssql" allows for multiple possible query parameter tokens,
	//   which is not recommended with SQL Server.
	// * Using "sqlserver" uses the native query parameter token, "@Name".
	conn, err := sql.Open("sqlserver", dsn)
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

	q := `IF OBJECT_ID(@p1, 'U') IS NULL
	CREATE TABLE ` + cleanedTableName + ` (
	migration_id VARCHAR(128) PRIMARY KEY NOT NULL,
	label VARCHAR(255) DEFAULT '',
	executed_at BIGINT DEFAULT 0
)`

	_, err = d.connection.ExecContext(ctx, q, cleanedTableName)
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
		q := `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = @p1`
		_, err = conn.ExecContext(ctx, q, version)
		return
	}

	// #nosec G202 -- table name was sanitized
	q := `INSERT INTO ` + cleanedTableName + ` (migration_id, label, executed_at) VALUES (@p1, @p2, @p3)`
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

	// In order to let existing data have a default value of '' or 0, add some named constraints.
	// For SQL Server, constraint names cannot have the quote characters, so remove them.
	//
	// The intent of adding a NULLABLE column but having a default value is to allow
	// for existing rows to be valid and to allow the godfish library to be simple
	// enough to not have to use sql.NullString or sql.NullInt64. And the intent of
	// that is to insulate the cassandra driver from having to know about database/sql.
	constraintPrefix := `DF_` + unquoteCleanedTablename(cleanedTableName)
	labelConstraint := constraintPrefix + `_label`
	executedAtConstraint := constraintPrefix + `_executed_at`

	// #nosec G202 -- table name was sanitized
	q := `ALTER TABLE ` + cleanedTableName + `
	ADD
		label VARCHAR(255) NULL
			CONSTRAINT ` + labelConstraint + ` DEFAULT '' WITH VALUES,
		executed_at BIGINT NULL
			CONSTRAINT ` + executedAtConstraint + ` DEFAULT 0 WITH VALUES;`
	_, xerr := tx.ExecContext(ctx, q)
	if xerr != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf(errMsgPrefix+", exec and rollback failed, exec error (%w), rollback error (%w) ", xerr, rerr)
		}
		return fmt.Errorf(errMsgPrefix+", exec failed but fortunately the rollback was OK; exec error %w", xerr)
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

func checkSchemaMigrationMetadata(ctx context.Context, d *Driver, tableName string) (out metadataResult, err error) {
	// Expect for the input tableName to have been treated by cleanIdentifier.
	// It doesn't need to be quoted in this case because it's used as a regular
	// query parameter in this function.
	tableName = unquoteCleanedTablename(tableName)

	lgr := slog.With("driver", d.Name(), slog.String("table_name", tableName))

	const query = `
SELECT t.table_name, c.column_name
FROM information_schema.tables t LEFT JOIN information_schema.columns c
    ON  t.table_name = c.table_name
    AND c.column_name IN (@p1, @p2)
WHERE t.table_catalog = DB_NAME()
	AND t.table_name = @p3
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

func unquoteCleanedTablename(in string) string {
	in = strings.ReplaceAll(in, quoteCharBegin, "")
	return strings.ReplaceAll(in, quoteCharEnd, "")
}

const quoteCharBegin, quoteCharEnd = `[`, `]`

func quotePart(part string) string { return quoteCharBegin + part + quoteCharEnd }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
