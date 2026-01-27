package cmd

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/rafaelespinoza/alf"
	"github.com/rafaelespinoza/godfish"
)

const upgradeCmdName = "upgrade"

func makeUpgradeSchemaMigrations(name string) alf.Directive {
	var timeout time.Duration

	return &alf.Command{
		Description: "add columns to schema migrations table",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags := newFlagSet(name)
			flags.DurationVar(
				&timeout,
				"timeout",
				0,
				fmt.Sprintf("max duration to run, ignored if non-positive, example vals %q", exampleDurationVals),
			)
			if f := p.Lookup("migrations-table"); f != nil {
				flags.StringVar(
					&commonArgs.MigrationsTable,
					f.Name,
					f.Value.String(),
					f.Usage,
				)
			}
			flags.Usage = func() {
				_, _ = fmt.Fprintf(flags.Output(), `Usage: %s [godfish-flags] %s [%s-flags]

	Add new columns to the schema migrations table.

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
	returns early without doing anything else.
`,
					bin, name, name)
				printFlagDefaults(&p)
				printFlagDefaults(flags)
			}
			return flags
		},
		Run: func(ctx context.Context) error {
			var cancel func()
			if timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			return withConnection(ctx, "", theDriver, func(ictx context.Context) error {
				return godfish.UpgradeSchemaMigrations(ictx, theDriver, commonArgs.MigrationsTable)
			})
		},
	}
}
