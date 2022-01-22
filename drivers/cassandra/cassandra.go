package cassandra

import (
	"regexp"
	"strings"

	"github.com/gocql/gocql"
	"github.com/rafaelespinoza/godfish"
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

func (d *driver) Execute(query string, args ...interface{}) (err error) {
	statements := statementDelimiter.Split(query, -1)
	if len(statements) < 1 {
		return
	}
	for _, q := range statements {
		if len(strings.TrimSpace(q)) < 1 {
			continue
		}
		err = d.connection.Query(q).Exec()
		if err != nil {
			return
		}
	}
	return nil
}

func (d *driver) CreateSchemaMigrationsTable() (err error) {
	err = d.connection.Query(
		`CREATE TABLE IF NOT EXISTS schema_migrations (migration_id TEXT PRIMARY KEY)`,
	).Exec()
	return
}

func (d *driver) AppliedVersions() (out godfish.AppliedVersions, err error) {
	query := d.connection.Query(
		`SELECT migration_id FROM schema_migrations`,
	)

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

func (d *driver) UpdateSchemaMigrations(forward bool, version string) (err error) {
	conn := d.connection
	if forward {
		err = conn.Query(`
			INSERT INTO schema_migrations (migration_id)
			VALUES (?)`,
			version,
		).Exec()
	} else {
		err = conn.Query(`
			DELETE FROM schema_migrations
			WHERE migration_id = ?`,
			version,
		).Exec()
	}
	return
}
