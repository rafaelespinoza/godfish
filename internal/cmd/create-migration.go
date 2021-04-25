package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/rafaelespinoza/alf"
	"github.com/rafaelespinoza/godfish"
)

func makeCreateMigration(name string) alf.Directive {
	var label string
	var reversible bool

	return &alf.Command{
		Description: "generate migration files",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags := flag.NewFlagSet(name, flag.ExitOnError)
			flags.StringVar(
				&label,
				"name",
				"",
				"label the migration, ie: create_foos_table, update_bars_qux",
			)
			flags.BoolVar(
				&reversible,
				"reversible",
				true,
				"create a reversible migration?",
			)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s [godfish-flags] %s [%s-flags]

	Generate migration files: one meant for the "forward" direction,
	another meant for "reverse". Optionally create a migration in the forward
	direction only by passing the flag "-reversible=false". The "name" flag has
	no effects other than on the generated filename. The output filename
	automatically has a "version". Timestamp format: %s.
`,
					bin, name, name, godfish.TimeFormat,
				)
				printFlagDefaults(&p)
				printFlagDefaults(flags)
			}
			return flags
		},
		Run: func(_ context.Context) error {
			migration, err := godfish.NewMigrationParams(label, reversible, commonArgs.Files)
			if err != nil {
				return err
			}
			return migration.GenerateFiles()
		},
	}
}
