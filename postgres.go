package godfish

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"
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

// String generates a data source name (or connection URL) based on the fields.
func (p PostgresParams) String() string {
	return fmt.Sprintf(
		"postgresql://%s:%s/%s?client_encoding=%s&sslmode=require",
		p.Host, p.Port, p.Name, p.Encoding,
	)
}

// postgres implements the Driver interface for postgres databases.
type postgres struct {
	MigrationsConf MigrationsConf
	connParams     PostgresParams
}

var _ Driver = (*postgres)(nil)

func newPostgres(connParams PostgresParams) (*postgres, error) {
	if connParams.Port == "" {
		connParams.Port = "5432"
	}
	driver := postgres{
		MigrationsConf: MigrationsConf{"db/migrations"},
		connParams:     connParams,
	}
	return &driver, nil
}

func (d *postgres) Name() string         { return "postgres" }
func (d *postgres) DSNParams() DSNParams { return d.connParams }

func (d *postgres) CreateSchemaMigrationsTable(conn *sql.DB) (err error) {
	_, err = conn.Query(
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

func (d *postgres) AppliedVersions(conn *sql.DB) (rows *sql.Rows, err error) {
	rows, err = conn.Query(
		`SELECT migration_id FROM schema_migrations ORDER BY migration_id ASC`,
	)
	return
}

func (d *postgres) UpdateSchemaMigrations(conn *sql.DB, dir Direction, version string) (err error) {
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
