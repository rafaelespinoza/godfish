package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"bitbucket.org/rafaelespinoza/godfish/godfish"
)

type command struct {
	// description should summarize the command in < 1 line.
	description string
	// setup should prepare Args for interpretation by using the pointer to Args
	// with the returned flag set.
	setup func(a *Args) *flag.FlagSet
	// run is a wrapper function that selects the necessary command line inputs,
	// executes the command and returns any errors.
	run func(a Args) error
}

// commands registers any operation by name to a command.
var commands = map[string]*command{
	"create-migration": &command{
		description: "generate migration files",
		setup: func(a *Args) *flag.FlagSet {
			flags := flag.NewFlagSet("create-migration", flag.ExitOnError)
			flags.StringVar(
				&a.Name,
				"name",
				"",
				"create a migration with a name, ie: foo",
			)
			flags.BoolVar(
				&a.Reversible,
				"reversible",
				true,
				"create a reversible migration?",
			)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s create-migration -name name [-reversible]

	Generate migration files: one meant for the "forward" direction,
	another meant for "reverse". Optionally create a migration in the forward
	direction only by passing the flag "-reversible=false". The "name" flag has
	no effects other than on the generated filename. The output filename
	automatically has a "version". Timestamp format: %s.`,
					bin, godfish.TimeFormat,
				)
				printFlagDefaults(flags)
			}
			return flags
		},
		run: func(a Args) error {
			dir, err := os.Open(a.Files)
			if err != nil {
				return err
			}
			migration, err := godfish.NewMigrationParams(a.Name, a.Reversible, dir)
			if err != nil {
				return err
			}
			return migration.GenerateFiles()
		},
	},
	"dump-schema": &command{
		description: "generate a sql file describing the db schema",
		setup: func(a *Args) *flag.FlagSet {
			flags := flag.NewFlagSet("dump-schema", flag.ExitOnError)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s dump-schema

	Print a database structure file to standard output.`,
					bin)
				printFlagDefaults(flags)
			}
			return flags
		},
		run: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			godfish.DumpSchema(driver)
			return nil
		},
	},
	"info": &command{
		description: "output current state of migrations",
		setup: func(a *Args) *flag.FlagSet {
			flags := flag.NewFlagSet("info", flag.ExitOnError)
			flags.StringVar(
				&a.Direction,
				"direction",
				"forward",
				"which way to look? (forward|reverse)",
			)
			flags.StringVar(
				&a.Version,
				"version",
				"",
				fmt.Sprintf("timestamp of migration, format: %s", godfish.TimeFormat),
			)
			flags.Usage = func() { // TODO: mention version
				fmt.Printf(`Usage: %s info [-direction forward|reverse] [-version version]

	Output info on helper functions.`, bin)
				printFlagDefaults(flags)
			}
			return flags
		},
		run: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			direction := whichDirection(a)
			return godfish.Info(driver, a.Files, direction, a.Version)
		},
	},
	"init": &command{
		description: "create godfish configuration file",
		setup: func(a *Args) *flag.FlagSet {
			flags := flag.NewFlagSet("init", flag.ExitOnError)
			flags.StringVar(
				&a.Conf,
				"conf",
				".godfish.json",
				"path to godfish config file",
			)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s init [-conf pathToFile]

	Creates a configuration file, unless it already exists.`, bin)
				printFlagDefaults(flags)
			}

			return flags
		},
		run: func(a Args) error {
			return godfish.Init(a.Conf)
		},
	},
	"migrate": &command{
		description: "execute migration(s) in the forward direction",
		setup: func(a *Args) *flag.FlagSet {
			flags := flag.NewFlagSet("migrate", flag.ExitOnError)
			flags.StringVar(
				&a.Version,
				"version",
				"",
				fmt.Sprintf("timestamp of migration, format: %s", godfish.TimeFormat),
			)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s migrate [-version timestamp]

	Execute migration(s) in the forward direction. If the "version" is left
	unspecified, then all available migrations are executed. Otherwise,
	available migrations are executed up to and including the specified version.
	Specify a version in the form: %s.`,
					bin, godfish.TimeFormat,
				)
				printFlagDefaults(flags)
			}

			return flags
		},
		run: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}

			err = godfish.Migrate(
				driver,
				a.Files,
				godfish.DirForward,
				a.Version,
			)
			return err
		},
	},
	"remigrate": &command{
		description: "rollback and then re-apply the last migration",
		setup: func(a *Args) *flag.FlagSet {
			flags := flag.NewFlagSet("remigrate", flag.ExitOnError)
			// TODO: files flag?
			flags.Usage = func() { // TODO: mention files flag?
				fmt.Printf(`Usage: %s remigrate

	Execute the last migration in reverse (rollback) and then execute the same
	one forward. This could be useful for development.`, bin)
				printFlagDefaults(flags)
			}

			return flags
		},
		run: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			if err = godfish.ApplyMigration(
				driver,
				a.Files,
				godfish.DirReverse,
				"",
			); err != nil {
				return err
			}
			return godfish.ApplyMigration(
				driver,
				a.Files,
				godfish.DirForward,
				"",
			)
		},
	},
	"rollback": &command{
		description: "execute migration(s) in the reverse direction",
		setup: func(a *Args) *flag.FlagSet {
			flags := flag.NewFlagSet("rollback", flag.ExitOnError)
			// TODO: files flag?
			flags.StringVar(
				&a.Version,
				"version",
				"",
				fmt.Sprintf("timestamp of migration, format: %s", godfish.TimeFormat),
			)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s rollback [-version timestamp]

	Execute migration(s) in the reverse direction. If the "version" is left
	unspecified, then only the first available migration is executed. Otherwise,
	available migrations are executed down to and including the specified
	version. Specify a version in the form: %s.`,
					bin, godfish.TimeFormat,
				)
				printFlagDefaults(flags)
			}
			return flags
		},
		run: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			if a.Version == "" {
				err = godfish.ApplyMigration(
					driver,
					a.Files,
					godfish.DirReverse,
					a.Version,
				)
			} else {
				err = godfish.Migrate(
					driver,
					a.Files,
					godfish.DirReverse,
					a.Version,
				)
			}
			return err
		},
	},
}

func newDriver(driverName string) (driver godfish.Driver, err error) {
	// try to keep the environment variable keys the same for all drivers.
	var (
		dbHost     = os.Getenv("DB_HOST")
		dbName     = os.Getenv("DB_NAME")
		dbPassword = os.Getenv("DB_PASSWORD")
		dbPort     = os.Getenv("DB_PORT")
		dbUser     = os.Getenv("DB_USER")
	)

	switch driverName {
	case "postgres":
		driver, err = godfish.NewDriver(godfish.PostgresParams{
			Encoding: "UTF8",
			Host:     dbHost,
			Name:     dbName,
			Pass:     dbPassword,
			Port:     dbPort,
			User:     dbUser,
		}, nil)
	case "mysql":
		driver, err = godfish.NewDriver(godfish.MySQLParams{
			Encoding: "UTF8",
			Host:     dbHost,
			Name:     dbName,
			Pass:     dbPassword,
			Port:     dbPort,
			User:     dbUser,
		}, nil)
	default:
		err = fmt.Errorf("unsupported db driver %q", driverName)
	}
	return
}

func whichDirection(a Args) (direction godfish.Direction) {
	direction = godfish.DirForward
	d := strings.ToLower(a.Direction)
	if strings.HasPrefix(d, "rev") || strings.HasPrefix(d, "back") {
		direction = godfish.DirReverse
	}
	return
}

// printFlagDefaults calls PrintDefaults on f. It helps make help message
// formatting more consistent.
func printFlagDefaults(f *flag.FlagSet) {
	fmt.Printf("\n\nFlags:\n\n")
	f.PrintDefaults()
}
