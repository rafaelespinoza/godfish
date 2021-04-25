package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rafaelespinoza/alf"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal/info"
)

func makeInfo(name string) alf.Directive {
	var direction, format, version string

	return &alf.Command{
		Description: "output applied migrations, migrations to apply",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags := flag.NewFlagSet(name, flag.ExitOnError)
			flags.StringVar(
				&direction,
				"direction",
				"forward",
				"which way to look? (forward|reverse)",
			)
			flags.StringVar(
				&format,
				"format",
				"tsv",
				"output format, one of (json|tsv)",
			)
			flags.StringVar(
				&version,
				"version",
				"",
				fmt.Sprintf("timestamp of migration, format: %s", godfish.TimeFormat),
			)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s [godfish-flags] %s [%s-flags]

	List applied migrations, preview migrations to apply.

	It's an introspection tool that can be used to show exactly which migration
	versions would be applied, in either a forward or reverse direction, before
	applying them.

	Migrations are categorized as:

	- up: Has been migrated against the DB.
	- down: Available to migrate, but hasn't yet.

	It also takes a "direction" flag if you want to know what would be applied
	in a rollback or remigrate operation. The "version" flag can be used to
	limit or extend the range of migrations to apply.
`,
					bin, name, name)
				printFlagDefaults(&p)
				printFlagDefaults(flags)
			}
			return flags
		},
		Run: func(_ context.Context) error {
			direction := whichDirection(direction)
			printer := choosePrinter(format, os.Stdout)
			return godfish.Info(theDriver, commonArgs.Files, direction, version, printer)
		},
	}
}

func whichDirection(input string) (direction godfish.Direction) {
	direction = godfish.DirForward
	d := strings.ToLower(input)
	if strings.HasPrefix(d, "rev") || strings.HasPrefix(d, "back") {
		direction = godfish.DirReverse
	}
	return
}

func choosePrinter(format string, w io.Writer) (printer godfish.InfoPrinter) {
	if format == "json" {
		printer = info.NewJSON(w)
		return
	}

	if format != "tsv" && format != "" {
		fmt.Fprintf(os.Stderr, "unknown format %q, defaulting to tsv\n", format)
	}
	printer = info.NewTSV(w)
	return
}
