package godfish

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
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

type Postgres struct {
	migrationConf MigrationsConf
	connParams    PGParams
	name          string
}

var _ Driver = (*Postgres)(nil)

func NewPostgres() (Driver, error) {
	port := "5432"
	if p := os.Getenv("DB_PORT"); p != "" {
		port = p
	}
	driver := Postgres{
		migrationConf: MigrationsConf{
			Table:       "schema_migrations",
			IDColumn:    "migration_id",
			PathToFiles: "db/migrations",
		},
		connParams: PGParams{
			Encoding: "UTF8",
			Host:     "localhost",
			Name:     os.Getenv("DB_NAME"),
			Pass:     os.Getenv("DB_PASSWORD"),
			Port:     port,
		},
	}
	return &driver, nil
}

func (d *Postgres) Name() string         { return "postgres" }
func (d *Postgres) DSNParams() DSNParams { return d.connParams }

func (d *Postgres) CreateSchemaMigrationsTable(conn *sql.DB) (err error) {
	// Cannot use arguments as identifiers in postgres, so we must use golang to
	// generate a query string before sending it to the DB connection. This is
	// normally not something we want to do b/c the sql library will escape
	// inputs when using the (query, ...args) form of Query, Exec, etc.
	q := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (%s VARCHAR(128) PRIMARY KEY NOT NULL)",
		d.migrationConf.Table,
		d.migrationConf.IDColumn,
	)
	_, err = conn.Query(q)
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

func (d *Postgres) ApplyMigration(conn *sql.DB, version string, dir Direction) (err error) {
	if _, err = conn.Exec(""); err != nil { // TODO: put in contents of migration
		return
	}

	if dir == DirForward {
		_, err = conn.Exec(
			`INSERT INTO "$1" ($2) VALUES $3`,
			d.migrationConf.Table,
			d.migrationConf.IDColumn,
			version,
		)
	} else {
		_, err = conn.Exec(
			`DELETE FROM "$1" WHERE "$2" = '$3'`,
			d.migrationConf.Table,
			d.migrationConf.IDColumn,
			version,
		)
	}
	return
}
