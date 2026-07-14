package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"

	"github.com/urfave/cli/v3"
)

func makeCreateMigration(subcmdName string, pathToConfig *string) *cli.Command {
	const fwdlabelFlagname, revlabelFlagname = "fwdlabel", "revlabel"

	return &cli.Command{
		Name:  subcmdName,
		Usage: "Generate migration files",
		Description: fmt.Sprintf(`Generate migration files: one meant for the "forward" direction,
another meant for "reverse". Optionally create a migration in the forward
direction only by passing the flag "-reversible=false". The "name" flag has
no effects other than on the generated filename. The output filename
automatically has a "version". Timestamp layout: %s.

Acceptable values for the %q and %q flags are:
	- %s
	- %s`,
			internal.TimeFormat,
			fwdlabelFlagname, revlabelFlagname,
			strings.Join(internal.ForwardDirections, ", "),
			strings.Join(internal.ReverseDirections, ", "),
		),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "name",
				Usage: "label the migration, ie: create_foos_table, update_bars_qux",
			},
			&cli.BoolFlag{
				Name:  "reversible",
				Value: true,
				Usage: "create a reversible migration?",
			},
			&cli.StringFlag{
				Name:    fwdlabelFlagname,
				Value:   internal.ForwardDirections[0],
				Usage:   "customize the directional part of the filename for forward migration",
				Sources: newSourceConfigChain(pathToConfig, "forward_label"),
			},
			&cli.StringFlag{
				Name:    revlabelFlagname,
				Value:   internal.ReverseDirections[0],
				Usage:   "customize the directional part of the filename for reverse migration",
				Sources: newSourceConfigChain(pathToConfig, "reverse_label"),
			},
		},
		Action: func(_ context.Context, c *cli.Command) error {
			migrationName := c.String("name")
			reversible := c.Bool("reversible")
			pathToFiles := c.String(pathToFilesFlagname)
			fwdlabelValue := c.String(fwdlabelFlagname)
			revlabelValue := c.String(revlabelFlagname)

			return godfish.CreateMigrationFiles(migrationName, reversible, pathToFiles, fwdlabelValue, revlabelValue)
		},
	}
}
