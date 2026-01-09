// Package cassandra provides a [godfish.Driver] for cassandra databases.
package cassandra

import (
	"context"
	"regexp"
	"strings"

	"github.com/gocql/gocql"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

// NewDriver creates a new cassandra driver.
func NewDriver() godfish.Driver { return &driver{} }

// driver implements the Driver interface for cassandra databases.
type driver struct {
	connection *gocql.Session
}

func (d *driver) Name() string { return "cassandra" }
func (d *driver) Connect(in string) (err error) {
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

func (d *driver) Close() (err error) {
	conn := d.connection
	if conn == nil {
		return
	}
	d.connection = nil
	conn.Close()
	return
}

var statementDelimiter = regexp.MustCompile(`;\s*\n`)

func (d *driver) Execute(ctx context.Context, query string, args ...any) (err error) {
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

func (d *driver) CreateSchemaMigrationsTable(ctx context.Context, migrationsTable string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	q := `CREATE TABLE IF NOT EXISTS ` + cleanedTableName + ` (migration_id TEXT PRIMARY KEY)`
	err = d.connection.Query(q).WithContext(ctx).Exec()
	return
}

func (d *driver) AppliedVersions(ctx context.Context, migrationsTable string) (out godfish.AppliedVersions, err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	q := `SELECT migration_id FROM ` + cleanedTableName
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

func (d *driver) UpdateSchemaMigrations(ctx context.Context, migrationsTable string, forward bool, version string) (err error) {
	cleanedTableName, err := cleanIdentifier(migrationsTable)
	if err != nil {
		return
	}

	conn := d.connection
	var q string
	if forward {
		q = `INSERT INTO ` + cleanedTableName + ` (migration_id) VALUES (?)`
	} else {
		q = `DELETE FROM ` + cleanedTableName + ` WHERE migration_id = ?`
	}
	err = conn.Query(q, version).WithContext(ctx).Exec()
	return
}

func quotePart(part string) string { return `"` + part + `"` }

func cleanIdentifier(input string) (string, error) {
	return internal.CleanNamespacedIdentifier(input, quotePart)
}
