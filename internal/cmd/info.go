package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/rafaelespinoza/alf"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

func makeInfo(name string) alf.Directive {
	var direction, format, version string
	var timeout time.Duration

	return &alf.Command{
		Description: "output applied migrations, migrations to apply",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags := newFlagSet(name)
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
				fmt.Sprintf("timestamp of migration, format: %s", internal.TimeFormat),
			)
			flags.DurationVar(
				&timeout,
				"timeout",
				0,
				fmt.Sprintf("max duration to run, ignored if non-positive, example vals %q", exampleDurationVals),
			)
			flags.Usage = func() {
				_, _ = fmt.Fprintf(flags.Output(), `Usage: %s [godfish-flags] %s [%s-flags]

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
		Run: func(ctx context.Context) error {
			var cancel func()
			if timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			dirFS := os.DirFS(commonArgs.Files)
			migrationsTable := commonArgs.MigrationsTable
			err := withConnection(ctx, "", theDriver, func(ictx context.Context) error {
				return godfish.Info(ictx, theDriver, dirFS.(fs.ReadDirFS), forward(direction), version, os.Stdout, format, migrationsTable)
			})
			if errors.Is(err, godfish.ErrSchemaMigrationsMissingColumns) {
				err = fmt.Errorf("%w; run the %q command to fix this", err, upgradeCmdName)
			}
			return err
		},
	}
}

func forward(input string) bool {
	d := strings.ToLower(input)
	for _, prefix := range []string{"rev", "roll", "back", "down"} {
		if strings.HasPrefix(d, prefix) {
			return false
		}
	}
	return true
}
