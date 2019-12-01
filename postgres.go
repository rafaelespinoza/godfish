package godfish

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os/exec"
)

// PGParams defines keys, values needed to connect to a postgres database.
type PGParams struct {
	Encoding string // Encoding is the client encoding for the connection.
	Host     string // Host is the name of the host to connect to.
	Name     string // Name is the database name.
	Pass     string // Pass is the password to use for the connection.
	Port     string // Port is the connection port.
	User     string // User is the name of the user to connect as.
}

var _ DSNParams = (*PGParams)(nil)

// String generates a data source name (or connection URL) based on the fields.
func (p PGParams) String() string {
	return fmt.Sprintf(
		"postgresql://%s:%s/%s?client_encoding=%s&sslmode=require",
		p.Host, p.Port, p.Name, p.Encoding,
	)
}

// Postgres implements the Driver interface for postgres databases.
type Postgres struct {
	MigrationsConf MigrationsConf
	connParams     PGParams
}

var _ Driver = (*Postgres)(nil)

func NewPostgres(connParams PGParams) (*Postgres, error) {
	if connParams.Port == "" {
		connParams.Port = "5432"
	}
	driver := Postgres{
		MigrationsConf: MigrationsConf{"db/migrations"},
		connParams:     connParams,
	}
	return &driver, nil
}

func (d *Postgres) Name() string         { return "postgres" }
func (d *Postgres) DSNParams() DSNParams { return d.connParams }

func (d *Postgres) CreateSchemaMigrationsTable(conn *sql.DB) (err error) {
	_, err = conn.Query(
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_id VARCHAR(128) PRIMARY KEY NOT NULL
		)`)
	return
}

func (d *Postgres) DumpSchema() (err error) {
	log.Println("dumping schema")
	var out bytes.Buffer
	cmd := exec.Command(
		"pg_dump",
		d.connParams.String(),
		"--schema-only", "--no-acl", "--no-owner", "--no-password",
		d.connParams.Name,
	)
	cmd.Stdout = &out
	if err = cmd.Run(); err != nil {
		return
	}
	log.Println(out.String())
	return
}

func (d *Postgres) AppliedVersions(conn *sql.DB) (rows *sql.Rows, err error) {
	rows, err = conn.Query(
		`SELECT migration_id FROM schema_migrations ORDER BY migration_id ASC`,
	)
	return
}

func (d *Postgres) UpdateSchemaMigrations(conn *sql.DB, dir Direction, version string) (err error) {
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
