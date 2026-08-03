// Package cassandra provides a [driver.Driver] for cassandra databases.
package cassandra

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/rafaelespinoza/godfish/driver"
	"github.com/rafaelespinoza/godfish/drivers/internal"

	"github.com/gocql/gocql"
)

const msgPrefix = "godfish/cassandra"
const driverMsgPrefix = msgPrefix + ".Driver"

// SampleDSN is an example data source name.
const SampleDSN = `cassandra://server_host:9042/keyspace_name?timeout_ms=2000&connect_timeout_ms=2000` // #nosec G101 -- not real credentials.

// NewDriver creates a new cassandra driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [driver.Driver] interface for cassandra databases.
type Driver struct {
	connection *gocql.Session
	keyspace   string
}

func (d *Driver) Name() string { return "cassandra" }
func (d *Driver) Connect(in string) error {
	if d.connection != nil {
		return nil
	}

	cluster, err := newClusterConfig(in)
	if err != nil {
		return fmt.Errorf("%s.%s: cluster config error: %w", driverMsgPrefix, "Connect", err)
	}
	d.keyspace = cluster.Keyspace
	conn, err := cluster.CreateSession()
	if err != nil {
		return fmt.Errorf("%s.%s: creating session: %w", driverMsgPrefix, "Connect", err)
	}
	d.connection = conn
	return nil
}

func (d *Driver) Close() error {
	conn := d.connection
	if conn == nil {
		return nil
	}
	d.connection = nil
	conn.Close()
	return nil
}

var statementDelimiter = regexp.MustCompile(`;\s*\n`)

func (d *Driver) Execute(ctx context.Context, query string, args ...any) error {
	statements := statementDelimiter.Split(query, -1)
	if len(statements) < 1 {
		return nil
	}
	for _, q := range statements {
		if len(strings.TrimSpace(q)) < 1 {
			continue
		}
		err := d.connection.Query(q).WithContext(ctx).Exec()
		if err != nil {
			return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "Execute", err)
		}
	}
	return nil
}

func (d *Driver) CreateSchemaMigrationsTable(ctx context.Context, migrationsTable string) error {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "CreateSchemaMigrationsTable", err)
	}

	q := `CREATE TABLE IF NOT EXISTS ` + cleanedTableName + ` (
	migration_id TEXT PRIMARY KEY,
	label TEXT,
	executed_at BIGINT
)`
	if err = d.connection.Query(q).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "CreateSchemaMigrationsTable", err)
	}
	return nil
}

func (d *Driver) AppliedVersions(ctx context.Context, migrationsTable string) (driver.AppliedVersions, error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: %w", driverMsgPrefix, "AppliedVersions", err)
	}

	metadata, err := checkKeyspaceMetadata(ctx, d, cleanedTableName)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: checking keyspace metadata: %w", driverMsgPrefix, "AppliedVersions", err)
	} else if !metadata.hasTable {
		err = fmt.Errorf("%s.%s: %w", driverMsgPrefix, "AppliedVersions", driver.ErrSchemaMigrationsDoesNotExist)
		return nil, err
	} else if !metadata.hasColLabel || !metadata.hasColExecutedAt {
		err = fmt.Errorf("%s.%s: %w", driverMsgPrefix, "AppliedVersions", driver.ErrSchemaMigrationsMissingColumns)
		return nil, err
	}

	q := `SELECT migration_id, label, executed_at FROM ` + cleanedTableName
	query := d.connection.Query(q).WithContext(ctx)
	slog.Debug(driverMsgPrefix+".AppliedVersions query",
		slog.String("keyspace", query.Keyspace()),
		slog.String("statement", query.Statement()),
	)

	av := execAllAscending(query)
	if av.closingErr == nil && av.scanningErr == nil {
		return av, nil
	}

	// An error here is probably more serious, prioritize that one if it exists.
	if av.closingErr != nil {
		slog.Error(driverMsgPrefix+".AppliedVersions non-empty error(s) after executing query",
			slog.Any("closing_err", av.closingErr),
			slog.Any("scanning_err", av.scanningErr), // just in case there's another lingering error...
		)
		return nil, fmt.Errorf("%s.%s: closing: %w", driverMsgPrefix, "AppliedVersions", av.closingErr)
	}

	slog.Error(driverMsgPrefix+".AppliedVersions non-empty scanning error",
		slog.Any("scanning_err", av.scanningErr),
		slog.String("type", fmt.Sprintf("%T", av.scanningErr)),
	)
	ierr, ok := av.scanningErr.(gocql.RequestError)
	if !ok {
		return nil, fmt.Errorf("%s.%s: scanning: %w", driverMsgPrefix, "AppliedVersions", av.scanningErr)
	}

	slog.Error(driverMsgPrefix+".AppliedVersions more details on the same scanning error",
		slog.String("type", fmt.Sprintf("%T", ierr)), slog.String("error", ierr.Error()),
		slog.Int("code", ierr.Code()), slog.String("message", ierr.Message()),
	)
	return nil, fmt.Errorf("%s.%s: scanning (this is a RequestError): %w", driverMsgPrefix, "AppliedVersions", ierr)
}

func (d *Driver) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version, label string) error {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "UpdateSchemaMigrations", err)
	}

	conn := d.connection
	if !forward {
		q := `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = ?`
		if err = conn.Query(q, version).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "UpdateSchemaMigrations", err)
		}
		return nil
	}

	q := `INSERT INTO ` + cleanedTableName + ` (migration_id, label, executed_at) VALUES (?, ?, ?)`
	now := time.Now().UTC()
	if err = conn.Query(q, version, label, now.Unix()).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "UpdateSchemaMigrations", err)
	}
	return nil
}

func (d *Driver) UpgradeSchemaMigrations(ctx context.Context, migrationsTable string) error {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "UpgradeSchemaMigrations", err)
	}

	// AFAIK, cassandra will not allow you to add multiple columns to the same
	// table within the same query. Add each column with its own statement and
	// await for each node in the cluster to be in agreement.
	type update struct{ columnName, query string }
	updates := make([]update, 0, 2)

	lgr := slog.With(slog.String("keyspace", d.keyspace), slog.String("table_name", cleanedTableName))
	startTime := time.Now()
	const timeSinceLogKey = "time_since_start_ms"
	defer func() { lgr.Info(driverMsgPrefix+": done", makeDurationMSAttr(timeSinceLogKey, startTime)) }()

	lgr.Debug(driverMsgPrefix+": starting check for keyspace metadata", makeDurationMSAttr(timeSinceLogKey, startTime))
	metadata, err := checkKeyspaceMetadata(ctx, d, cleanedTableName)
	if err != nil {
		return fmt.Errorf("%s.%s: checking keyspace metadata: %w", driverMsgPrefix, "UpgradeSchemaMigrations", err)
	} else if !metadata.hasTable {
		return fmt.Errorf("%s.%s: %w", driverMsgPrefix, "UpgradeSchemaMigrations", driver.ErrSchemaMigrationsDoesNotExist)
	}
	// Conditionally add the updates in case there's a need to retry 1 of them.
	if metadata.hasColLabel {
		lgr.Debug(driverMsgPrefix+".UpgradeSchemaMigrations: column appears to already exist, skipping", slog.String("column", "label"))
	} else {
		updates = append(
			updates,
			update{columnName: "label", query: `ALTER TABLE ` + cleanedTableName + ` ADD label TEXT`},
		)
	}
	if metadata.hasColExecutedAt {
		lgr.Debug(driverMsgPrefix+".UpgradeSchemaMigrations: column appears to already exist, skipping", slog.String("column", "executed_at"))
	} else {
		updates = append(
			updates,
			update{columnName: "executed_at", query: `ALTER TABLE ` + cleanedTableName + ` ADD executed_at BIGINT`},
		)
	}
	lgr.Debug(driverMsgPrefix+".UpgradeSchemaMigrations: updates prepared", slog.Int("num_updates", len(updates)))
	for i, u := range updates {
		ulgr := lgr.With(slog.Int("i", i), slog.String("column", u.columnName))
		ulgr.Info(driverMsgPrefix+".UpgradeSchemaMigrations: starting upgrade query", makeDurationMSAttr(timeSinceLogKey, startTime))
		if err = d.connection.Query(u.query).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf(
				driverMsgPrefix+".UpgradeSchemaMigrations: upgrading schema migrations table for column %s: %w",
				u.columnName, err,
			)
		}
		ulgr.Info(driverMsgPrefix+".UpgradeSchemaMigrations: query complete, now awaiting schema agreement...", makeDurationMSAttr(timeSinceLogKey, startTime))
		// Make sure all nodes know about the new column.
		if err = d.connection.AwaitSchemaAgreement(ctx); err != nil {
			return fmt.Errorf("%s.%s: awaiting schema agreement after adding column: %s", driverMsgPrefix, "UpgradeSchemaMigrations", u.columnName)
		}
		ulgr.Info(driverMsgPrefix+".UpgradeSchemaMigrations: cluster is in agreement", makeDurationMSAttr(timeSinceLogKey, startTime))
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
		lgr.Debug(msgPrefix+": checked keyspace metadata",
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
		msgPrefix+": checking for table existence",
		slog.String("query", tableQuery), slog.Any("args", tableArgs),
	)
	tableScanner := d.connection.Query(tableQuery, tableArgs...).WithContext(ctx).Iter().Scanner()
	defer func() {
		// The Err method also releases resources. The scanner should not be
		// used after this point.
		cerr := tableScanner.Err()
		if cerr != nil {
			lgr.Error(msgPrefix+": closing table query scanner", slog.Any("error", cerr))
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
		msgPrefix+": checking for column existence",
		slog.String("query", columnsQuery), slog.Any("args", colArgs),
	)

	colScanner := d.connection.Query(columnsQuery, colArgs...).WithContext(ctx).Iter().Scanner()
	defer func() {
		// The Err method also releases resources. The scanner should not be
		// used after this point.
		cerr := colScanner.Err()
		if cerr != nil {
			lgr.Error(msgPrefix+": closing column query scanner", slog.Any("error", cerr))
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
