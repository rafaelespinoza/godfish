package godfish_test

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/drivers/mysql"
	"github.com/rafaelespinoza/godfish/drivers/postgres"
	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
	"github.com/rafaelespinoza/godfish/drivers/sqlserver"
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
	ctx := context.Background()
	var (
		err           error
		migrationsDir fs.FS
	)

	// dsn is a database-specific connection string (data source name). In this
	// sqlite3 example, it's a path to a file.
	dsn := filepath.Join(os.TempDir(), "godfish_test.sqlite")
	driver := sqlite3.NewDriver()
	if err = driver.Connect(dsn); err != nil {
		fmt.Println("connecting to DB", err)
		return
	}
	defer func() { _ = driver.Close() }()

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
	if err = godfish.Migrate(ctx, driver, migrationsDir, forward, "3456", migrationsTable); err != nil {
		fmt.Println("migrating DB", err)
		return
	}

	// Show the state of the DB migrations as TSV (default).
	if err = godfish.Info(ctx, driver, migrationsDir, forward, "", os.Stdout, "tsv", migrationsTable); err != nil {
		fmt.Println("getting, showing info", err)
		return
	}
}

// How to migrate to apply all available migrations.
func ExampleMigrate_migrateToLatestVersion() {
	ctx := context.Background()

	// driver can be one of the drivers in this project, see drivers/.
	driver := mysql.NewDriver()
	if err := driver.Connect(mysql.SampleDSN); err != nil {
		fmt.Println("connecting to DB", err)
		return
	}
	defer func() { _ = driver.Close() }()

	migrationsDir := os.DirFS("path/to/migration/files")

	// version may match the "version" part of the migration filename
	// ie: YYYYMMDDHHmmss. When version is empty, then this function will apply
	// all available migrations.
	var version string

	// migrationsTable when empty, will use the default value, "schema_migrations".
	// This is where versioning is kept.
	var migrationsTable string

	err := godfish.Migrate(ctx, driver, migrationsDir, true, version, migrationsTable)
	if err != nil {
		fmt.Printf("attempting to migrate to latest version: %s", err.Error())
		return
	}
	fmt.Println("ok")
}

// How to rollback one migration.
func ExampleMigrate_rollbackOne() {
	ctx := context.Background()

	// driver can be one of the drivers in this project, see drivers/.
	driver := postgres.NewDriver()
	if err := driver.Connect(postgres.SampleDSN); err != nil {
		fmt.Println("connecting to DB", err)
		return
	}
	defer func() { _ = driver.Close() }()

	migrationsDir := os.DirFS("path/to/migration/files")

	// version when empty, tells this function to apply the nearest available
	// migration with a lower version than the current.
	// In other words, you rollback by 1 migration.
	var version string

	// migrationsTable when empty, will use the default value, "schema_migrations".
	// This is where versioning is kept.
	var migrationsTable string

	err := godfish.Migrate(ctx, driver, migrationsDir, false, version, migrationsTable)
	if err != nil {
		fmt.Printf("attempting to migrate to latest version: %s", err.Error())
		return
	}
	fmt.Println("ok")
}

// How to rollback 1 migration and reapply it.
func ExampleApplyMigration_remigrate() {
	ctx := context.Background()

	// driver can be one of the drivers in this project, see drivers/.
	driver := sqlserver.NewDriver()
	if err := driver.Connect(sqlserver.SampleDSN); err != nil {
		fmt.Println("connecting to DB", err)
		return
	}
	defer func() { _ = driver.Close() }()

	migrationsDir := os.DirFS("path/to/migration/files")

	// migrationsTable when empty, will use the default value, "schema_migrations".
	// This is where versioning is kept.
	var migrationsTable string

	if err := godfish.ApplyMigration(ctx, driver, migrationsDir, false, "", migrationsTable); err != nil {
		fmt.Printf("attempting to rollback during remigration operation: %s", err.Error())
		return
	}
	if err := godfish.ApplyMigration(ctx, driver, migrationsDir, true, "", migrationsTable); err != nil {
		fmt.Printf("attempting to migrate forward during remigration operation: %s", err.Error())
		return
	}
	fmt.Println("ok")
}
