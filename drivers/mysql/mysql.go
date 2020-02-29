package mysql

import (
	"database/sql"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	my "github.com/go-sql-driver/mysql"
	"github.com/rafaelespinoza/godfish"
)

// DSN implements the godfish.DSN interface and defines keys, values needed to
// connect to a mysql database.
type DSN struct {
	godfish.ConnectionParams
}

var _ godfish.DSN = (*DSN)(nil)

// Boot initializes the DSN from environment inputs.
func (p *DSN) Boot(params godfish.ConnectionParams) error {
	p.ConnectionParams = params
	return nil
}

// NewDriver creates a new mysql driver.
func (p *DSN) NewDriver(migConf *godfish.MigrationsConf) (godfish.Driver, error) {
	return newMySQL(*p)
}

// String generates a data source name (or connection URL) based on the fields.
func (p *DSN) String() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s",
		p.User, p.Pass, p.Host, p.Port, p.Name,
	)
}

// driver implements the godfish.Driver interface for mysql databases.
type driver struct {
	connection *sql.DB
	dsn        DSN
}

var _ godfish.Driver = (*driver)(nil)

func newMySQL(dsn DSN) (*driver, error) {
	if dsn.Host == "" {
		dsn.Host = "localhost"
	}
	if dsn.Port == "" {
		dsn.Port = "3306"
	}
	return &driver{dsn: dsn}, nil
}

func (d *driver) Name() string     { return "mysql" }
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

var statementDelimiter = regexp.MustCompile(`;\s*\n`)

func (d *driver) Execute(query string, args ...interface{}) (err error) {
	// Attempt to support migrations with 1 or more statements. AFAIK, the
	// standard library does not support executing multiple statements at once.
	// As a workaround, break them up and apply them.
	statements := statementDelimiter.Split(query, -1)
	if len(statements) < 1 {
		return
	}
	tx, err := d.connection.Begin()
	if err != nil {
		return
	}
	for _, q := range statements {
		if len(strings.TrimSpace(q)) < 1 {
			continue
		}
		_, err = tx.Exec(q)
		if err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				return fmt.Errorf("%w; %v", err, rerr)
			}
			return
		}
	}
	return tx.Commit()
}

func (d *driver) CreateSchemaMigrationsTable() (err error) {
	_, err = d.connection.Exec(
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_id VARCHAR(128) PRIMARY KEY NOT NULL
		)`)
	return
}

func (d *driver) DumpSchema() (err error) {
	params := d.dsn
	cmd := exec.Command(
		"mysqldump",
		"--user", params.User, "--password="+params.Pass, // skip password prompt by a omitting space
		"--host", params.Host, "--port", params.Port,
		"--comments", "--no-data", "--routines", "--triggers", "--tz-utc",
		"--skip-add-drop-table", "--add-locks", "--create-options", "--set-charset",
		params.Name,
	)

	out, err := cmd.Output()
	if val, ok := err.(*exec.ExitError); ok {
		fmt.Println(string(val.Stderr))
		err = val
		return
	}
	fmt.Println(string(out))
	return
}

func (d *driver) AppliedVersions() (out godfish.AppliedVersions, err error) {
	rows, err := d.connection.Query(
		`SELECT migration_id FROM schema_migrations ORDER BY migration_id ASC`,
	)
	if ierr, ok := err.(*my.MySQLError); ok {
		// https://dev.mysql.com/doc/refman/8.0/en/server-error-reference.html#error_er_no_such_table
		if ierr.Number == 1146 {
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
			VALUES (?)`,
			version,
		)
	} else {
		_, err = conn.Exec(`
			DELETE FROM schema_migrations
			WHERE migration_id = ?`,
			version,
		)
	}
	return
}
