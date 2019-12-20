package godfish

import (
	"database/sql"
	"fmt"
	"os/exec"

	"github.com/go-sql-driver/mysql"
)

// MySQLParams implements the DSNParams interface and defines keys, values
// needed to connect to a mysql database.
type MySQLParams struct {
	Encoding string // Encoding is the client encoding for the connection.
	Host     string // Host is the name of the host to connect to.
	Name     string // Name is the database name.
	Pass     string // Pass is the password to use for the connection.
	Port     string // Port is the connection port.
	User     string // User is the name of the user to connect as.
}

var _ DSNParams = (*MySQLParams)(nil)

func (p MySQLParams) NewDriver(migConf *MigrationsConf) (Driver, error) {
	return newMySQL(p)
}

// String generates a data source name (or connection URL) based on the fields.
func (p MySQLParams) String() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s",
		p.User, p.Pass, p.Host, p.Port, p.Name,
	)
}

// my implements the Driver interface for mysql databases.
type my struct {
	connection *sql.DB
	connParams MySQLParams
}

var _ Driver = (*my)(nil)

func newMySQL(connParams MySQLParams) (*my, error) {
	if connParams.Host == "" {
		connParams.Host = "localhost"
	}
	if connParams.Port == "" {
		connParams.Port = "3306"
	}
	driver := my{
		connParams: connParams,
	}
	return &driver, nil
}

func (d *my) Name() string         { return "mysql" }
func (d *my) DSNParams() DSNParams { return d.connParams }
func (d *my) Connect() (conn *sql.DB, err error) {
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

func (d *my) Close() (err error) {
	conn := d.connection
	if conn == nil {
		return
	}
	d.connection = nil
	err = conn.Close()
	return
}

func (d *my) Execute(query string, args ...interface{}) (err error) {
	_, err = d.connection.Query(query)
	return
}

func (d *my) CreateSchemaMigrationsTable() (err error) {
	_, err = d.connection.Query(
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_id VARCHAR(128) PRIMARY KEY NOT NULL
		)`)
	return
}

func (d *my) DumpSchema() (err error) {
	params := d.connParams
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

func (d *my) AppliedVersions() (out AppliedVersions, err error) {
	rows, err := d.connection.Query(
		`SELECT migration_id FROM schema_migrations ORDER BY migration_id ASC`,
	)
	if ierr, ok := err.(*mysql.MySQLError); ok {
		// https://dev.mysql.com/doc/refman/8.0/en/server-error-reference.html#error_er_no_such_table
		if ierr.Number == 1146 {
			err = ErrSchemaMigrationsDoesNotExist
		}
	}
	out = AppliedVersions(rows)
	return
}

func (d *my) UpdateSchemaMigrations(dir Direction, version string) (err error) {
	conn := d.connection
	if dir == DirForward {
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
