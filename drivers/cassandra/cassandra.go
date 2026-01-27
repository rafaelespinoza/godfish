// Package cassandra provides a [godfish.Driver] for cassandra databases.
package cassandra

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"

	"github.com/gocql/gocql"
)

const msgPrefix = "cassandra: "

// NewDriver creates a new cassandra driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [godfish.Driver] interface for cassandra databases.
type Driver struct {
	connection *gocql.Session
	keyspace   string
}

func (d *Driver) Name() string { return "cassandra" }
func (d *Driver) Connect(in string) (err error) {
	if d.connection != nil {
		return
	}

	cluster, err := newClusterConfig(in)
	if err != nil {
		return
	}
	d.keyspace = cluster.Keyspace
	conn, err := cluster.CreateSession()
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
	conn.Close()
	return
}

var statementDelimiter = regexp.MustCompile(`;\s*\n`)

func (d *Driver) Execute(ctx context.Context, query string, args ...any) (err error) {
	statements := statementDelimiter.Split(query, -1)
	if len(statements) < 1 {
		return
	}
	for _, q := range statements {
		if len(strings.TrimSpace(q)) < 1 {
			continue
		}
		err = d.connection.Query(q).WithContext(ctx).Exec()
		if err != nil {
			return
		}
	}
	return nil
}

func (d *Driver) CreateSchemaMigrationsTable(ctx context.Context, migrationsTable string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	q := `CREATE TABLE IF NOT EXISTS ` + cleanedTableName + ` (
	migration_id TEXT PRIMARY KEY,
	label TEXT,
	executed_at BIGINT
)`
	err = d.connection.Query(q).WithContext(ctx).Exec()
	return
}

func (d *Driver) AppliedVersions(ctx context.Context, migrationsTable string) (out godfish.AppliedVersions, err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	metadata, err := checkKeyspaceMetadata(ctx, d, cleanedTableName)
	if err != nil {
		return
	} else if !metadata.hasTable {
		err = godfish.ErrSchemaMigrationsDoesNotExist
		return
	} else if !metadata.hasColLabel || !metadata.hasColExecutedAt {
		err = godfish.ErrSchemaMigrationsMissingColumns
		return
	}

	q := `SELECT migration_id, label, executed_at FROM ` + cleanedTableName
	query := d.connection.Query(q).WithContext(ctx)
	slog.Debug(msgPrefix+"(*driver).AppliedVersions query",
		slog.String("keyspace", query.Keyspace()),
		slog.String("statement", query.Statement()),
	)

	av := execAllAscending(query)
	if av.closingErr == nil && av.scanningErr == nil {
		out = av
		return
	}

	// An error here is probably more serious, prioritize that one if it exists.
	if av.closingErr != nil {
		slog.Error(msgPrefix+"(*driver).AppliedVersions non-empty error(s) after executing query",
			slog.Any("closing_err", av.closingErr),
			slog.Any("scanning_err", av.scanningErr), // just in case there's another lingering error...
		)
		err = av.closingErr
		return
	}

	slog.Error(msgPrefix+"(*driver).AppliedVersions non-empty scanning error",
		slog.Any("scanning_err", av.scanningErr),
		slog.String("type", fmt.Sprintf("%T", av.scanningErr)),
	)
	ierr, ok := av.scanningErr.(gocql.RequestError)
	if !ok {
		err = av.scanningErr
		return
	}

	slog.Error(msgPrefix+"(*driver).AppliedVersions more details on the same scanning error",
		slog.String("type", fmt.Sprintf("%T", ierr)), slog.String("error", ierr.Error()),
		slog.Int("code", ierr.Code()), slog.String("message", ierr.Message()),
	)
	err = ierr
	return
}

func (d *Driver) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version, label string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	conn := d.connection
	if !forward {
		q := `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = ?`
		err = conn.Query(q, version).WithContext(ctx).Exec()
		return
	}

	q := `INSERT INTO ` + cleanedTableName + ` (migration_id, label, executed_at) VALUES (?, ?, ?)`
	now := time.Now().UTC()
	err = conn.Query(q, version, label, now.Unix()).WithContext(ctx).Exec()
	return
}

func (d *Driver) UpgradeSchemaMigrations(ctx context.Context, migrationsTable string) error {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return err
	}

	// AFAIK, cassandra will not allow you to add multiple columns to the same
	// table within the same query. Add each column with its own statement and
	// await for each node in the cluster to be in agreement.
	type update struct{ columnName, query string }
	updates := make([]update, 0, 2)

	lgr := slog.With(slog.String("keyspace", d.keyspace), slog.String("table_name", cleanedTableName))
	startTime := time.Now()
	const timeSinceLogKey = "time_since_start_ms"
	defer func() { lgr.Info(msgPrefix+"done", makeDurationMSAttr(timeSinceLogKey, startTime)) }()

	lgr.Debug(msgPrefix+"starting check for keyspace metadata", makeDurationMSAttr(timeSinceLogKey, startTime))
	metadata, err := checkKeyspaceMetadata(ctx, d, cleanedTableName)
	if err != nil {
		return err
	} else if !metadata.hasTable {
		return godfish.ErrSchemaMigrationsDoesNotExist
	}
	// Conditionally add the updates in case there's a need to retry 1 of them.
	if metadata.hasColLabel {
		lgr.Debug(msgPrefix+"column appears to already exist, skipping", slog.String("column", "label"))
	} else {
		updates = append(
			updates,
			update{columnName: "label", query: `ALTER TABLE ` + cleanedTableName + ` ADD label TEXT`},
		)
	}
	if metadata.hasColExecutedAt {
		lgr.Debug(msgPrefix+"column appears to already exist, skipping", slog.String("column", "executed_at"))
	} else {
		updates = append(
			updates,
			update{columnName: "executed_at", query: `ALTER TABLE ` + cleanedTableName + ` ADD executed_at BIGINT`},
		)
	}
	lgr.Debug(msgPrefix+"updates prepared", slog.Int("num_updates", len(updates)))
	for i, u := range updates {
		ulgr := lgr.With(slog.Int("i", i), slog.String("column", u.columnName))
		ulgr.Info(msgPrefix+"starting upgrade query", makeDurationMSAttr(timeSinceLogKey, startTime))
		if err = d.connection.Query(u.query).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf(
				msgPrefix+"upgrading schema migrations table for column %s; %w",
				u.columnName, err,
			)
		}
		ulgr.Info(msgPrefix+"query complete, now awaiting schema agreement...", makeDurationMSAttr(timeSinceLogKey, startTime))
		// Make sure all nodes know about the new column.
		if err = d.connection.AwaitSchemaAgreement(ctx); err != nil {
			return fmt.Errorf(msgPrefix+"awaiting schema agreement after adding column %s", u.columnName)
		}
		ulgr.Info(msgPrefix+"cluster is in agreement", makeDurationMSAttr(timeSinceLogKey, startTime))
	}

	return nil
}

func makeDurationMSAttr(key string, startedAt time.Time) slog.Attr {
	dur := time.Since(startedAt)
	return slog.Int64(key, dur.Milliseconds())
}

type metadataResult struct {
	hasTable         bool
	hasColLabel      bool
	hasColExecutedAt bool
}

// checkKeyspaceMetadata inspects the schema of the schema_migrations table
// within the current keyspace.
func checkKeyspaceMetadata(ctx context.Context, d *Driver, tableName string) (out metadataResult, err error) {
	// Expect for the input tableName to have been treated by cleanIdentifier.
	// It doesn't need to be quoted in this case because it's used as a regular
	// query parameter in this function.
	tableName = strings.ReplaceAll(tableName, quote, "")

	lgr := slog.With(slog.String("driver", d.Name()), slog.String("keyspace", d.keyspace), slog.String("table_name", tableName))

	defer func() {
		lgr.Debug(msgPrefix+"checked keyspace metadata",
			slog.Group("result",
				slog.Bool("has_table", out.hasTable),
				slog.Bool("has_col_label", out.hasColLabel),
				slog.Bool("has_col_executed_at", out.hasColExecutedAt),
			),
		)
	}()

	const tableQuery = `SELECT table_name FROM system_schema.tables WHERE keyspace_name = ? AND table_name = ?`
	tableArgs := []any{d.keyspace, tableName}
	lgr.Debug(
		msgPrefix+"checking for table existence",
		slog.String("query", tableQuery), slog.Any("args", tableArgs),
	)
	tableScanner := d.connection.Query(tableQuery, tableArgs...).WithContext(ctx).Iter().Scanner()
	defer func() {
		// The Err method also releases resources. The scanner should not be
		// used after this point.
		cerr := tableScanner.Err()
		if cerr != nil {
			lgr.Error(msgPrefix+"closing table query scanner", slog.Any("error", cerr))
		}
	}()
	for tableScanner.Next() {
		out.hasTable = true
	}

	if !out.hasTable {
		return
	}

	const columnsQuery = `
SELECT table_name, column_name
FROM system_schema.columns
WHERE keyspace_name = ?
	AND table_name = ?
	AND column_name IN ?`
	colArgs := []any{d.keyspace, tableName, []string{"label", "executed_at"}}
	lgr.Debug(
		msgPrefix+"checking for column existence",
		slog.String("query", columnsQuery), slog.Any("args", colArgs),
	)

	colScanner := d.connection.Query(columnsQuery, colArgs...).WithContext(ctx).Iter().Scanner()
	defer func() {
		// The Err method also releases resources. The scanner should not be
		// used after this point.
		cerr := colScanner.Err()
		if cerr != nil {
			lgr.Error(msgPrefix+"closing column query scanner", slog.Any("error", cerr))
		}
	}()
	for colScanner.Next() {
		out.hasTable = true
		var t, c string
		if err = colScanner.Scan(&t, &c); err != nil {
			err = fmt.Errorf("scanning for columns; %w", err)
			return
		}
		switch c {
		case "label":
			out.hasColLabel = true
		case "executed_at":
			out.hasColExecutedAt = true
		}
	}

	return
}

const quote = `"`

func quotePart(part string) string { return quote + part + quote }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
