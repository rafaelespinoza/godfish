package postgres

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"

	"bitbucket.org/rafaelespinoza/godfish/godfish"
	"github.com/lib/pq"
)

// Params implements the godfish.DSNParams interface and defines keys, values
// needed to connect to a postgres database.
type Params struct {
	Encoding string // Encoding is the client encoding for the connection.
	Host     string // Host is the name of the host to connect to.
	Name     string // Name is the database name.
	Pass     string // Pass is the password to use for the connection.
	Port     string // Port is the connection port.
	User     string // User is the name of the user to connect as.
}

var _ godfish.DSNParams = (*Params)(nil)

// NewDriver creates a new postgres driver.
func (p Params) NewDriver(migConf *godfish.MigrationsConf) (godfish.Driver, error) {
	return newPostgres(p)
}

// String generates a data source name (or connection URL) based on the fields.
func (p Params) String() string {
	return fmt.Sprintf(
		"postgresql://%s:%s/%s?client_encoding=%s&sslmode=require",
		p.Host, p.Port, p.Name, p.Encoding,
	)
}

// driver implements the Driver interface for postgres databases.
type driver struct {
	connection *sql.DB
	connParams Params
}

var _ godfish.Driver = (*driver)(nil)

func newPostgres(connParams Params) (*driver, error) {
	if connParams.Host == "" {
		connParams.Host = "localhost"
	}
	if connParams.Port == "" {
		connParams.Port = "5432"
	}
	return &driver{connParams: connParams}, nil
}

func (d *driver) Name() string                 { return "postgres" }
func (d *driver) DSNParams() godfish.DSNParams { return d.connParams }
func (d *driver) Connect() (conn *sql.DB, err error) {
	if d.connection != nil {
		conn = d.connection
		return
	}
	if conn, err = sql.Open(d.Name(), d.DSNParams().String()); err != nil {
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
		d.connParams.Name,
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
