package cmd

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/rafaelespinoza/alf"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

func makeCreateMigration(subcmdName string) alf.Directive {
	const fwdlabelFlagname, revlabelFlagname = "fwdlabel", "revlabel"
	var migrationName, fwdlabelValue, revlabelValue string
	var reversible bool

	// Other subcommands scope the flagset within the Setup func. However, this
	// one is scoped up here to check if some flags were specified at runtime.
	flags := newFlagSet(subcmdName)

	return &alf.Command{
		Description: "generate migration files",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags.StringVar(
				&migrationName,
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
			flags.StringVar(
				&fwdlabelValue,
				fwdlabelFlagname,
				internal.ForwardDirections[0],
				"customize the directional part of the filename for forward migration",
			)
			flags.StringVar(
				&revlabelValue,
				revlabelFlagname,
				internal.ReverseDirections[0],
				"customize the directional part of the filename for reverse migration",
			)
			flags.Usage = func() {
				fmt.Fprintf(flags.Output(), `Usage: %s [godfish-flags] %s [%s-flags]

	Generate migration files: one meant for the "forward" direction,
	another meant for "reverse". Optionally create a migration in the forward
	direction only by passing the flag "-reversible=false". The "name" flag has
	no effects other than on the generated filename. The output filename
	automatically has a "version". Timestamp layout: %s.

	Acceptable values for the %q and %q flags are:
	- %s
	- %s
`,
					bin, subcmdName, subcmdName, internal.TimeFormat,
					fwdlabelFlagname, revlabelFlagname,
					strings.Join(internal.ForwardDirections, ", "), strings.Join(internal.ReverseDirections, ", "),
				)
				printFlagDefaults(&p)
				printFlagDefaults(flags)
			}
			return flags
		},
		Run: func(_ context.Context) error {
			// Allow this subcommand's flags to override names for directional
			// part of the filename. But allow for the configuration file to
			// have a say if the flag wasn't passed in at runtime.
			var passedFwd, passedRev bool
			flags.Visit(func(f *flag.Flag) {
				switch f.Name {
				case fwdlabelFlagname:
					passedFwd = true
				case revlabelFlagname:
					passedRev = true
				default:
					break
				}
			})
			if !passedFwd && commonArgs.DefaultFwdLabel != "" {
				fwdlabelValue = commonArgs.DefaultFwdLabel
			}
			if !passedRev && commonArgs.DefaultRevLabel != "" {
				revlabelValue = commonArgs.DefaultRevLabel
			}

			return godfish.CreateMigrationFiles(migrationName, reversible, commonArgs.Files, fwdlabelValue, revlabelValue)
		},
	}
}
