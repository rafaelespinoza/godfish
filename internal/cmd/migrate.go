package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/rafaelespinoza/godfish"
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

	if errors.Is(err, godfish.ErrSchemaMigrationsMissingColumns) {
		err = fmt.Errorf("%w; run the %q command to fix this", err, upgradeCmdName)
	}
	return err
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

	if errors.Is(err, godfish.ErrSchemaMigrationsMissingColumns) {
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

	var rollbackFn func(context.Context, godfish.Driver, fs.FS, ...godfish.Opter) error
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

	if errors.Is(err, godfish.ErrSchemaMigrationsMissingColumns) {
		err = fmt.Errorf("%w; run the %q command to fix this", err, upgradeCmdName)
	}
	return err
}
