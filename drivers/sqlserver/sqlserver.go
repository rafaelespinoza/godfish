package sqlserver

import (
	"database/sql"
	"errors"
	"strings"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/rafaelespinoza/godfish"
)

// NewDriver creates a new Microsoft SQL Server driver.
func NewDriver() godfish.Driver { return &driver{} }

// driver implements the godfish.Driver interface for Microsoft SQL Server.
type driver struct {
	connection *sql.DB
}

func (d *driver) Name() string { return "sqlserver" }
func (d *driver) Connect(dsn string) (err error) {
	if d.connection != nil {
		return
	}

	// github.com/denisenkom/go-mssqldb registers two sql.Driver names:
	// "mssql" and "sqlserver". Their docs seem to steer users towards the
	// latter, so just use that one.
	conn, err := sql.Open("sqlserver", dsn)
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
	err = conn.Close()
	return
}

func (d *driver) Execute(query string, args ...interface{}) (err error) {
	_, err = d.connection.Exec(query)
	return
}

func (d *driver) CreateSchemaMigrationsTable() (err error) {
	_, err = d.connection.Exec(`
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.tables WHERE table_schema = (SELECT schema_name()) AND table_name = 'schema_migrations'
		)
		CREATE TABLE schema_migrations (migration_id VARCHAR(128) PRIMARY KEY NOT NULL)
	`)
	return
}

func (d *driver) AppliedVersions() (out godfish.AppliedVersions, err error) {
	rows, err := d.connection.Query(`SELECT migration_id FROM schema_migrations ORDER BY migration_id ASC`)

	var ierr mssql.Error
	// https://docs.microsoft.com/en-us/sql/relational-databases/errors-events/database-engine-events-and-errors
	// Invalid object name 'schema_migrations'
	if errors.As(err, &ierr) && ierr.SQLErrorNumber() == 208 && strings.Contains(ierr.Error(), "schema_migrations") {
		err = godfish.ErrSchemaMigrationsDoesNotExist
	}
	out = godfish.AppliedVersions(rows)
	return
}

func (d *driver) UpdateSchemaMigrations(forward bool, version string) (err error) {
	conn := d.connection
	if forward {
		_, err = conn.Exec(`
			INSERT INTO schema_migrations (migration_id)
			VALUES (@p1)`,
			version,
		)
	} else {
		_, err = conn.Exec(`
			DELETE FROM schema_migrations
			WHERE migration_id = @p1`,
			version,
		)
	}
	return
}
