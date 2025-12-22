package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/driver"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/compat"

	"github.com/urfave/cli/v3"
)

func makeMigrate(name string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: "Execute migration(s) in the forward direction",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "version",
				Value: "",
				Usage: fmt.Sprintf("timestamp of migration, format: %s", internal.TimeFormat),
			},
			&cli.DurationFlag{
				Name:  timeoutFlagname,
				Value: 0,
				Usage: fmt.Sprintf("max duration to run, ignored if non-positive, example vals %q", exampleDurationVals),
			},
		},
		Description: fmt.Sprintf(`Execute migration(s) in the forward direction. If the "version" is left
unspecified, then all available migrations are executed. Otherwise,
available migrations are executed up to and including the specified version.
Specify a version in the form: %s.

The "files" flag can specify the path to a directory with migration files.`,
			internal.TimeFormat,
		),
		Action: func(ctx context.Context, c *cli.Command) error {
			driver, err := getDriver(ctx)
			if err != nil {
				return fmt.Errorf("getting driver from %s command: %w", name, err)
			}
			timeout := c.Duration(timeoutFlagname)
			dirFS := os.DirFS(c.String(pathToFilesFlagname))

			return runMigrate(ctx, driver, timeout, dirFS, compat.MigrationOptParams{
				TargetVersion:   c.String("version"),
				MigrationsTable: c.String(migrationsTableFlagname),
			})
		},
	}
}

func runMigrate(ctx context.Context, driverConn DriverConnector, timeout time.Duration, dirFS fs.FS, migOpts compat.MigrationOptParams) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	err := withConnection(ctx, "", driverConn, func(ictx context.Context) error {
		opts := compat.MakeMigrationOpts(migOpts)
		return godfish.MigrateWith(ictx, driverConn, dirFS, opts...)
	})

	if errors.Is(err, driver.ErrSchemaMigrationsMissingColumns) {
		err = fmt.Errorf("%w; run the %q command to fix this", err, upgradeCmdName)
	}
	return err
}

func makeApplyMigration(name string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: "run 1 migration, regardless of order",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "version",
				Value: "",
				Usage: fmt.Sprintf("timestamp of migration, format: %s", internal.TimeFormat),
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Value: 0,
				Usage: fmt.Sprintf("max duration to run, ignored if non-positive, example vals %q", exampleDurationVals),
			},
		},
		Description: `Run one migration matching the input version, regardless of its order.

For example,
* You already have these migrations applied: 1.
* Alice cuts a branch and adds a migration 2, but it's not merged.
* Bob works on another branch, adds migration 3, and it's merged into main.
* You run migration 3 on the production DB. Now migrations 1, 3 are applied.
* Alice, who was working on migration 2, gets it merged into main.
* You try to run the migrations, using the migrate subcommand.
  Nothing happens, why?

The schema migrations table sees that migration 3 has been applied. The tool
looks for a migration with a version greater than 3, doesn't see one, so it
returns early. Migration 2 is there, but it is not considered available for
migration because it's "out-of-order". How to apply migration 2?

There are a few options:
* Take Alice's migration, 2, and change its filename to so that it's lexically
  greater than the filename of 3. The migration would be considered available.
* Alternatively, you can apply any one migration with this subcommand by
  specifying the -version, even those that are out-of-order.

Use the info subcommand to view the state of the schema migrations table.

As mentioned for other subcommands, the "files" flag can specify the path to a
directory with migration files.`,
		Action: func(ctx context.Context, c *cli.Command) error {
			version := c.String("version")
			if version == "" {
				return errors.New("version is required")
			}

			driver, err := getDriver(ctx)
			if err != nil {
				return fmt.Errorf("getting driver from %s command: %w", name, err)
			}
			timeout := c.Duration("timeout")
			dirFS := os.DirFS(c.String(pathToFilesFlagname))

			return runApplyMigration(ctx, driver, timeout, dirFS, compat.MigrationOptParams{
				TargetVersion:   c.String("version"),
				MigrationsTable: c.String(migrationsTableFlagname),
			})
		},
	}
}

func runApplyMigration(ctx context.Context, driverConn DriverConnector, timeout time.Duration, dirFS fs.FS, migOpts compat.MigrationOptParams) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return withConnection(ctx, "", driverConn, func(ictx context.Context) error {
		opts := compat.MakeMigrationOpts(migOpts)
		return godfish.ApplyMigrationWith(ictx, driverConn, dirFS, opts...)
	})
}

func makeRemigrate(name string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: "Rollback and then re-apply the last migration",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:  timeoutFlagname,
				Value: 0,
				Usage: fmt.Sprintf("max duration to run, ignored if non-positive, example vals %q", exampleDurationVals),
			},
		},
		Description: `Execute the last migration in reverse (rollback) and then execute the same
one forward. This could be useful for development.

The "files" flag can specify the path to a directory with migration files.`,
		Action: func(ctx context.Context, c *cli.Command) error {
			driver, err := getDriver(ctx)
			if err != nil {
				return fmt.Errorf("getting driver from %s command: %w", name, err)
			}
			timeout := c.Duration(timeoutFlagname)
			dirFS := os.DirFS(c.String(pathToFilesFlagname))
			migOpts := compat.MigrationOptParams{MigrationsTable: c.String(migrationsTableFlagname)}

			return runRemigrate(ctx, driver, timeout, dirFS, migOpts)
		},
	}
}

func runRemigrate(ctx context.Context, driverConn DriverConnector, timeout time.Duration, dirFS fs.FS, migOpts compat.MigrationOptParams) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	err := withConnection(ctx, "", driverConn, func(ictx context.Context) error {
		opts := compat.MakeMigrationOpts(migOpts)
		if ierr := godfish.ApplyRollbackWith(ctx, driverConn, dirFS, opts...); ierr != nil {
			return ierr
		}
		return godfish.ApplyMigrationWith(ictx, driverConn, dirFS, opts...)
	})

	if errors.Is(err, driver.ErrSchemaMigrationsMissingColumns) {
		err = fmt.Errorf("%w; run the %q command to fix this", err, upgradeCmdName)
	}
	return err
}

func makeRollback(name string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: "Execute migration(s) in the reverse direction",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "version",
				Value: "",
				Usage: fmt.Sprintf("timestamp of migration, format: %s", internal.TimeFormat),
			},
			&cli.DurationFlag{
				Name:  timeoutFlagname,
				Value: 0,
				Usage: fmt.Sprintf("max duration to run, ignored if non-positive, example vals %q", exampleDurationVals),
			},
		},
		Description: fmt.Sprintf(`Execute migration(s) in the reverse direction. If the "version" is left
unspecified, then only the first available migration is executed. Otherwise,
available migrations are executed down to and including the specified
version. Specify a version in the form: %s.

The "files" flag can specify the path to a directory with migration files.`,
			internal.TimeFormat),
		Action: func(ctx context.Context, c *cli.Command) error {
			driver, err := getDriver(ctx)
			if err != nil {
				return fmt.Errorf("getting driver from %s command: %w", name, err)
			}
			timeout := c.Duration(timeoutFlagname)
			dirFS := os.DirFS(c.String(pathToFilesFlagname))

			return runRollback(ctx, driver, timeout, dirFS, compat.MigrationOptParams{
				MigrationsTable: c.String(migrationsTableFlagname),
				TargetVersion:   c.String("version"),
			})
		},
	}
}

func runRollback(ctx context.Context, driverConn DriverConnector, timeout time.Duration, dirFS fs.FS, migOpts compat.MigrationOptParams) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var rollbackFn func(context.Context, driver.Driver, fs.FS, ...godfish.Opter) error
	if migOpts.TargetVersion == "" {
		rollbackFn = godfish.ApplyRollbackWith
	} else {
		rollbackFn = godfish.RollbackWith
	}

	err := withConnection(ctx, "", driverConn, func(ictx context.Context) error {
		return rollbackFn(
			ictx,
			driverConn,
			dirFS,
			compat.MakeMigrationOpts(migOpts)...,
		)
	})

	if errors.Is(err, driver.ErrSchemaMigrationsMissingColumns) {
		err = fmt.Errorf("%w; run the %q command to fix this", err, upgradeCmdName)
	}
	return err
}
