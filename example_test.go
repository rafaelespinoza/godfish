package godfish_test

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
)

// migrationsFS is the embedded readonly file system.
// Its source is a relative directory.
//
//go:embed testdata/default
var migrationsFS embed.FS

// Demonstrate sqlite3 driver with embedded migrations data.
//
//	# The testdata to embed:
//	$ ls -1 testdata/default
//	forward-1234-alpha.sql
//	forward-2345-bravo.sql
//	forward-3456-charlie.sql
//	reverse-1234-alpha.sql
//	reverse-2345-bravo.sql
//	reverse-3456-charlie.sql
func Example_embed() {
	var (
		err           error
		migrationsDir fs.FS
	)

	// To connect to the DB with this library set the environment variable, DB_DSN.
	dbDSN := filepath.Join(os.TempDir(), "godfish_test.sqlite")
	if err = os.Setenv("DB_DSN", dbDSN); err != nil {
		fmt.Println("setting env var DB_DSN", err)
		return
	}
	driver := sqlite3.NewDriver()

	// Use fs.Sub to reference a subdirectory of the embedded files.
	if migrationsDir, err = fs.Sub(migrationsFS, "testdata/default"); err != nil {
		fmt.Println("getting fs subtree", err)
		return
	}

	// migrationsTable may be used to specify the table for recording DB migration state.
	// If empty, then the library will set it to a default value, "schema_migrations".
	var migrationsTable string

	// Apply the "forward" migrations through version "3456".
	forward := true
	if err = godfish.Migrate(driver, migrationsDir, forward, "3456", migrationsTable); err != nil {
		fmt.Println("migrating DB", err)
		return
	}

	// Show the state of the DB migrations as TSV (default).
	if err = godfish.Info(driver, migrationsDir, forward, "", os.Stdout, "tsv", migrationsTable); err != nil {
		fmt.Println("getting, showing info", err)
		return
	}
	// Output:
	// up	1234	forward-1234-alpha.sql
	// up	2345	forward-2345-bravo.sql
	// up	3456	forward-3456-charlie.sql
}
