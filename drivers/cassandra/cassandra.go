// Package cassandra provides a [godfish.Driver] for cassandra databases.
package cassandra

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

// NewDriver creates a new cassandra driver.
func NewDriver() *Driver { return &Driver{} }

// Driver implements the [godfish.Driver] interface for cassandra databases.
type Driver struct {
	connection *gocql.Session
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

	q := `SELECT migration_id, label, executed_at FROM ` + cleanedTableName
	query := d.connection.Query(q).WithContext(ctx)

	av := execAllAscending(query)

	if av.err == nil {
		out = av
		return
	}

	ierr, ok := av.err.(gocql.RequestError)
	if !ok {
		err = av.err
		return
	}

	// In cassandra v3, the error message might be "unconfigured table".
	// In cassandra v4, the error message might be "table does not exist".
	// Either version would return this error code (0x2200). At this time, the
	// gocql library does not seem to have a more specific error code.
	if ierr.Code() == gocql.ErrCodeInvalid && strings.Contains(ierr.Message(), "table") {
		err = godfish.ErrSchemaMigrationsDoesNotExist
	} else {
		err = av.err
	}

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
		q = `INSERT INTO ` + cleanedTableName + ` (migration_id, label, executed_at) VALUES (?, ?, ?)`
		now := time.Now().UTC()
		err = conn.Query(q, version, label, now.Unix()).WithContext(ctx).Exec()
	} else {
		q = `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = ?`
		err = conn.Query(q, version).WithContext(ctx).Exec()
	}
	return
}

func quotePart(part string) string { return `"` + part + `"` }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
