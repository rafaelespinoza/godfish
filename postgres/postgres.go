package postgres

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"

	"bitbucket.org/rafaelespinoza/godfish"
	"github.com/lib/pq"
)

// DSN implements the godfish.DSN interface and defines keys, values needed to
// connect to a postgres database.
type DSN struct {
	godfish.ConnectionParams
}

var _ godfish.DSN = (*DSN)(nil)

// Boot initializes the DSN from environment inputs.
func (p *DSN) Boot(params godfish.ConnectionParams) error {
	p.ConnectionParams = params
	return nil
}

// NewDriver creates a new postgres driver.
func (p *DSN) NewDriver(migConf *godfish.MigrationsConf) (godfish.Driver, error) {
	return newPostgres(*p)
}

// String generates a data source name (or connection URL) based on the fields.
func (p *DSN) String() string {
	return fmt.Sprintf(
		"postgresql://%s:%s/%s?client_encoding=%s&sslmode=require",
		p.Host, p.Port, p.Name, p.Encoding,
	)
}

// driver implements the Driver interface for postgres databases.
type driver struct {
	connection *sql.DB
	dsn        DSN
}

var _ godfish.Driver = (*driver)(nil)

func newPostgres(dsn DSN) (*driver, error) {
	if dsn.Host == "" {
		dsn.Host = "localhost"
	}
	if dsn.Port == "" {
		dsn.Port = "5432"
	}
	return &driver{dsn: dsn}, nil
}

func (d *driver) Name() string     { return "postgres" }
func (d *driver) DSN() godfish.DSN { return &d.dsn }
func (d *driver) Connect() (conn *sql.DB, err error) {
	if d.connection != nil {
		conn = d.connection
		return
	}
	if conn, err = sql.Open(d.Name(), d.DSN().String()); err != nil {
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
	_, err = d.connection.Query(query)
	return
}

func (d *driver) CreateSchemaMigrationsTable() (err error) {
	_, err = d.connection.Query(
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_id VARCHAR(128) PRIMARY KEY NOT NULL
		)`)
	return
}

func (d *driver) DumpSchema() (err error) {
	var out bytes.Buffer
	cmd := exec.Command(
		"pg_dump",
		"--schema-only", "--no-acl", "--no-owner", "--no-password",
		d.dsn.Name,
	)
	cmd.Stdout = &out
	if err = cmd.Run(); err != nil {
		return
	}
	fmt.Println(out.String())
	return
}

func (d *driver) AppliedVersions() (out godfish.AppliedVersions, err error) {
	rows, err := d.connection.Query(
		`SELECT migration_id FROM schema_migrations ORDER BY migration_id ASC`,
	)
	if ierr, ok := err.(*pq.Error); ok {
		if ierr.Message == "relation \"schema_migrations\" does not exist" {
			err = godfish.ErrSchemaMigrationsDoesNotExist
		}
	}
	out = godfish.AppliedVersions(rows)
	return
}

func (d *driver) UpdateSchemaMigrations(dir godfish.Direction, version string) (err error) {
	conn := d.connection
	if dir == godfish.DirForward {
		_, err = conn.Exec(`
			INSERT INTO schema_migrations (migration_id)
			VALUES ($1)
			RETURNING migration_id`,
			version,
		)
	} else {
		_, err = conn.Exec(`
			DELETE FROM schema_migrations
			WHERE migration_id = $1
			RETURNING migration_id`,
			version,
		)
	}
	return
}
