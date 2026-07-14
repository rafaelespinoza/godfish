package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/rafaelespinoza/godfish"

	"github.com/urfave/cli/v3"
)

const upgradeCmdName = "upgrade"

func makeUpgradeSchemaMigrations(name string, pathToConfig *string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: "Add columns to schema migrations table",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:  "timeout",
				Value: 0,
				Usage: fmt.Sprintf("max duration to run, ignored if non-positive, example vals %q", exampleDurationVals),
			},
			&cli.StringFlag{
				Name:  migrationsTableFlagname,
				Usage: "name of DB table for storing migration state",
				// Local is set to true to tell the cli library to not use the global flag
				// of the same name.
				Local:   true,
				Sources: newSourceConfigChain(pathToConfig, "migrations_table"),
			},
		},
		Description: `Add new columns to the schema migrations table.

If your schema migrations table was created with v0.14.0 or lower, then
the columns in that table are likely something like:

	migration_id VARCHAR PRIMARY KEY NOT NULL

This subcommand adds metadata columns to that table. The exact definition
will vary based on the DB driver, but it's roughly:

	label VARCHAR DEFAULT ''
	executed_at INT DEFAULT 0

The flag, -migrations-table, specifies which table to work on.
If that table does not exist yet, then there should be no need to use
this subcommand. If that table has already been upgraded, then it
returns early without doing anything else.`,
		Action: func(ctx context.Context, c *cli.Command) error {
			timeout := c.Duration("timeout")
			migrationsTable := c.String(migrationsTableFlagname)

			return runUpgrade(ctx, theDriver, timeout, migrationsTable)
		},
	}
}

func runUpgrade(ctx context.Context, driverConn DriverConnector, timeout time.Duration, migrationsTable string) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return withConnection(ctx, "", driverConn, func(ictx context.Context) error {
		return godfish.UpgradeSchemaMigrations(ictx, driverConn, migrationsTable)
	})
}
