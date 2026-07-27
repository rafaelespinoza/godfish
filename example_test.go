package godfish_test

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/drivers/cassandra"
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

	// Migrate all the way to the latest version.
	err = godfish.MigrateWith(ctx, driver, migrationsDir)
	if err != nil {
		fmt.Println("migrating DB", err)
		return
	}

	// Show the state of the DB migrations as TSV (default).
	if err = godfish.InfoWith(ctx, driver, migrationsDir); err != nil {
		fmt.Println("getting, showing info", err)
		return
	}
}

// Run one or more migrations in the forward direction.
func ExampleMigrateWith() {
	ctx := context.Background()
	var err error

	// driver can be one of the drivers in this project, see drivers/.
	driver := cassandra.NewDriver()
	if err := driver.Connect(mysql.SampleDSN); err != nil {
		fmt.Println("connecting to DB", err)
		return
	}
	defer func() { _ = driver.Close() }()

	// migrationsDir is an fs.FS directory with the migrations files.
	migrationsDir := os.DirFS("path/to/migration/files")

	// This will apply all available migrations in the forward direction,
	// relative to the current.
	err = godfish.MigrateWith(ctx, driver, migrationsDir)
	if err != nil {
		// Handle error
	}

	// version may match the "version" part of the migration filename,
	// ie: YYYYMMDDHHmmss. Each migration greater than the current, and up to and
	// including the target migration version, is applied.
	version := godfish.WithTargetVersion("20380119031408")

	err = godfish.MigrateWith(ctx, driver, migrationsDir, version)
	if err != nil {
		// Handle error
	}

	// migrationsTable may be changed from its default, "schema_migrations".
	// This table is where versioning is kept.
	migrationsTable := godfish.WithMigrationsTable("migration_versions")

	err = godfish.MigrateWith(ctx, driver, migrationsDir, migrationsTable)
	if err != nil {
		// Handle error
	}
}

// Run one or more rollbacks.
// In this library, these are considered to have a reverse direction.
func ExampleRollbackWith() {
	ctx := context.Background()
	var err error

	// driver can be one of the drivers in this project, see drivers/.
	driver := postgres.NewDriver()
	if err := driver.Connect(postgres.SampleDSN); err != nil {
		fmt.Println("connecting to DB", err)
		return
	}
	defer func() { _ = driver.Close() }()

	// migrationsDir is an fs.FS directory with the migrations files.
	migrationsDir := os.DirFS("path/to/migration/files")

	// This will apply all available rollback migrations.
	err = godfish.RollbackWith(ctx, driver, migrationsDir)
	if err != nil {
		// Handle error
	}

	// version may match the "version" part of the migration filename,
	// ie: YYYYMMDDHHmmss. This will apply all migrations between the current and
	// the targeted version.
	version := godfish.WithTargetVersion("19700101000000")

	err = godfish.RollbackWith(ctx, driver, migrationsDir, version)
	if err != nil {
		// Handle error
	}
}

// Run any one migration in the forward direction.
func ExampleApplyMigrationWith() {
	ctx := context.Background()
	var err error

	// driver can be one of the drivers in this project, see drivers/.
	driver := mysql.NewDriver()
	if err := driver.Connect(mysql.SampleDSN); err != nil {
		fmt.Println("connecting to DB", err)
		return
	}
	defer func() { _ = driver.Close() }()

	// migrationsDir is an fs.FS directory with the migrations files.
	migrationsDir := os.DirFS("path/to/migration/files")

	// This will apply the next available migration in the forward direction,
	// relative to the current.
	err = godfish.ApplyMigrationWith(ctx, driver, migrationsDir)
	if err != nil {
		// Handle error
	}

	// version may match the "version" part of the migration filename,
	// ie: YYYYMMDDHHmmss. Migrations between the current and the targeted version
	// are not applied.
	version := godfish.WithTargetVersion("20380119031408")

	err = godfish.ApplyMigrationWith(ctx, driver, migrationsDir, version)
	if err != nil {
		// Handle error
	}

	// migrationsTable may be changed from its default, "schema_migrations".
	// This table is where versioning is kept.
	migrationsTable := godfish.WithMigrationsTable("migration_versions")

	err = godfish.ApplyMigrationWith(ctx, driver, migrationsDir, migrationsTable)
	if err != nil {
		// Handle error
	}
}

// Run any one rollback migration.
// In this library, these are considered to have a reverse direction.
func ExampleApplyRollbackWith() {
	ctx := context.Background()
	var err error

	// driver can be one of the drivers in this project, see drivers/.
	driver := sqlserver.NewDriver()
	if err := driver.Connect(sqlserver.SampleDSN); err != nil {
		fmt.Println("connecting to DB", err)
		return
	}
	defer func() { _ = driver.Close() }()

	// migrationsDir is an fs.FS directory with the migrations files.
	migrationsDir := os.DirFS("path/to/migration/files")

	// This will apply the closest available rollback migration relative to the
	// current.
	err = godfish.ApplyRollbackWith(ctx, driver, migrationsDir)
	if err != nil {
		// Handle error
	}

	// version may match the "version" part of the migration filename,
	// ie: YYYYMMDDHHmmss. Migrations between the current and the targeted version
	// are not applied.
	version := godfish.WithTargetVersion("19700101000000")

	err = godfish.ApplyRollbackWith(ctx, driver, migrationsDir, version)
	if err != nil {
		// Handle error
	}
}
