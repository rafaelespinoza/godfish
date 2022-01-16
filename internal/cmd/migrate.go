package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/rafaelespinoza/alf"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

func makeMigrate(name string) alf.Directive {
	var version string

	return &alf.Command{
		Description: "execute migration(s) in the forward direction",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags := flag.NewFlagSet(name, flag.ExitOnError)
			flags.StringVar(
				&version,
				"version",
				"",
				fmt.Sprintf("timestamp of migration, format: %s", internal.TimeFormat),
			)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s [godfish-flags] %s [%s-flags]

	Execute migration(s) in the forward direction. If the "version" is left
	unspecified, then all available migrations are executed. Otherwise,
	available migrations are executed up to and including the specified version.
	Specify a version in the form: %s.

	The "files" flag can specify the path to a directory with migration files.
`,
					bin, name, name, internal.TimeFormat,
				)
				printFlagDefaults(&p)
				printFlagDefaults(flags)
			}

			return flags
		},
		Run: func(_ context.Context) error {
			err := godfish.Migrate(
				theDriver,
				commonArgs.Files,
				true,
				version,
			)
			return err
		},
	}
}

func makeRemigrate(name string) alf.Directive {
	return &alf.Command{
		Description: "rollback and then re-apply the last migration",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags := flag.NewFlagSet(name, flag.ExitOnError)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s [godfish-flags] %s [%s-flags]

	Execute the last migration in reverse (rollback) and then execute the same
	one forward. This could be useful for development.

	The "files" flag can specify the path to a directory with migration files.
`,
					bin, name, name)
				printFlagDefaults(&p)
				printFlagDefaults(flags)
			}

			return flags
		},
		Run: func(_ context.Context) error {
			err := godfish.ApplyMigration(theDriver, commonArgs.Files, false, "")
			if err != nil {
				return err
			}
			return godfish.ApplyMigration(theDriver, commonArgs.Files, true, "")
		},
	}
}

func makeRollback(name string) alf.Directive {
	var version string

	return &alf.Command{
		Description: "execute migration(s) in the reverse direction",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags := flag.NewFlagSet(name, flag.ExitOnError)
			flags.StringVar(
				&version,
				"version",
				"",
				fmt.Sprintf("timestamp of migration, format: %s", internal.TimeFormat),
			)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s [godfish-flags] %s [%s-flags]

	Execute migration(s) in the reverse direction. If the "version" is left
	unspecified, then only the first available migration is executed. Otherwise,
	available migrations are executed down to and including the specified
	version. Specify a version in the form: %s.

	The "files" flag can specify the path to a directory with migration files.
`,
					bin, name, name, internal.TimeFormat,
				)
				printFlagDefaults(&p)
				printFlagDefaults(flags)
			}
			return flags
		},
		Run: func(_ context.Context) error {
			var err error
			if version == "" {
				err = godfish.ApplyMigration(
					theDriver,
					commonArgs.Files,
					false,
					version,
				)
			} else {
				err = godfish.Migrate(
					theDriver,
					commonArgs.Files,
					false,
					version,
				)
			}
			return err
		},
	}
}
