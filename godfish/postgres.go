package godfish

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"

	"github.com/lib/pq"
)

// PostgresParams implements the DSNParams interface and defines keys, values
// needed to connect to a postgres database.
type PostgresParams struct {
	Encoding string // Encoding is the client encoding for the connection.
	Host     string // Host is the name of the host to connect to.
	Name     string // Name is the database name.
	Pass     string // Pass is the password to use for the connection.
	Port     string // Port is the connection port.
	User     string // User is the name of the user to connect as.
}

var _ DSNParams = (*PostgresParams)(nil)

func (p PostgresParams) NewDriver(migConf *MigrationsConf) (Driver, error) {
	return newPostgres(p)
}

// String generates a data source name (or connection URL) based on the fields.
func (p PostgresParams) String() string {
	return fmt.Sprintf(
		"postgresql://%s:%s/%s?client_encoding=%s&sslmode=require",
		p.Host, p.Port, p.Name, p.Encoding,
	)
}

// postgres implements the Driver interface for postgres databases.
type postgres struct {
	connection *sql.DB
	connParams PostgresParams
}

var _ Driver = (*postgres)(nil)

func newPostgres(connParams PostgresParams) (*postgres, error) {
	if connParams.Host == "" {
		connParams.Host = "localhost"
	}
	if connParams.Port == "" {
		connParams.Port = "5432"
	}
	driver := postgres{
		connParams: connParams,
	}
	return &driver, nil
}

func (d *postgres) Name() string         { return "postgres" }
func (d *postgres) DSNParams() DSNParams { return d.connParams }
func (d *postgres) Connect() (conn *sql.DB, err error) {
	if d.connection != nil {
		conn = d.connection
		return
	}
	if conn, err = connect(d.Name(), d.DSNParams()); err != nil {
		return
	}
	d.connection = conn
	return
}

func (d *postgres) Close() (err error) {
	conn := d.connection
	if conn == nil {
		return
	}
	d.connection = nil
	err = conn.Close()
	return
}

func (d *postgres) Execute(query string, args ...interface{}) (err error) {
	_, err = d.connection.Query(query)
	return
}

func (d *postgres) CreateSchemaMigrationsTable() (err error) {
	_, err = d.connection.Query(
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_id VARCHAR(128) PRIMARY KEY NOT NULL
		)`)
	return
}

func (d *postgres) DumpSchema() (err error) {
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

func (d *postgres) AppliedVersions() (out AppliedVersions, err error) {
	rows, err := d.connection.Query(
		`SELECT migration_id FROM schema_migrations ORDER BY migration_id ASC`,
	)
	if ierr, ok := err.(*pq.Error); ok {
		if ierr.Message == "relation \"schema_migrations\" does not exist" {
			err = ErrSchemaMigrationsDoesNotExist
		}
	}
	out = AppliedVersions(rows)
	return
}

func (d *postgres) UpdateSchemaMigrations(dir Direction, version string) (err error) {
	conn := d.connection
	if dir == DirForward {
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
