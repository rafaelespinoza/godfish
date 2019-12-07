package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"bitbucket.org/rafaelespinoza/godfish/godfish"
)

// Args describes the CLI inputs.
type Args struct {
	Cmd        string
	Direction  string
	Driver     string
	Help       bool
	Name       string
	Reversible bool
	Version    string
}

// argsToCommand is a wrapper function that selects and maps command line inputs
// needed by the associated command.
type argsToCommand func(Args) error

type command struct {
	description string // summarize in < 1 line.
	help        func() string
	mapper      argsToCommand
}

// commands registers any operation by name to a command.
var commands = map[string]command{
	"create-migration": {
		description: "generate migration files",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: %s create-migration -name <name> [-reversible]

	Generate migration files at %s: one meant for the "forward" direction,
	another meant for "reverse". Optionally create a migration in the forward
	direction only by passing the flag "-reversible=false". The "name" flag has
	no effects other than on the generated filename. The output filename
	automatically has a "version". Timestamp format: %s.`,
				bin, pathToDBMigrations, godfish.TimeFormat)
			return
		},
		mapper: func(a Args) error {
			dir, err := os.Open(pathToDBMigrations)
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
	"dump-schema": {
		description: "generate a sql file describing the db schema",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: %s dump-schema -driver <driverName>
	`, bin)
			return
		},
		mapper: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			godfish.DumpSchema(driver)
			return nil
		},
	},
	"info": {
		description: "output current state of migrations",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: %s info -direction [forward|reverse] -driver <driverName>
			`, bin)
			return
		},
		mapper: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			direction := whichDirection(a)
			return godfish.Info(driver, pathToDBMigrations, direction, a.Version)
		},
	},
	"init": {
		description: "create schema migrations table",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: %s init -driver <driverName>

Creates the db table to track migrations, unless it already exists.`, bin)
			return
		},
		mapper: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			return godfish.CreateSchemaMigrationsTable(driver)
		},
	},
	"migrate": {
		description: "execute migration(s) in the forward direction",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: %s migrate -driver <driverName> [-version <timestamp>]

	Execute migration(s) in the forward direction. If the "version" is left
	unspecified, then all available migrations are executed. Otherwise,
	available migrations are executed up to and including the specified version.
	Specify a version in the form: %s.`,
				bin, godfish.TimeFormat)
			return
		},
		mapper: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			if a.Version == "" {
				err = godfish.Migrate(
					driver,
					pathToDBMigrations,
					godfish.DirForward,
					a.Version,
				)
			} else {
				err = godfish.ApplyMigration(
					driver,
					pathToDBMigrations,
					godfish.DirForward,
					a.Version,
				)
			}
			return err
		},
	},
	"remigrate": {
		description: "rollback and then re-apply the last migration",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: %s remigrate -driver <driverName>

	Execute the last migration in reverse (rollback) and then execute the same
	one forward. This could be useful for development.`, bin)
			return
		},
		mapper: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			if err = godfish.ApplyMigration(
				driver,
				pathToDBMigrations,
				godfish.DirReverse,
				"",
			); err != nil {
				return err
			}
			return godfish.ApplyMigration(
				driver,
				pathToDBMigrations,
				godfish.DirForward,
				"",
			)
		},
	},
	"rollback": {
		description: "execute migration(s) in the reverse direction",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: %s rollback -driver <driverName> [-version <timestamp>]

	Execute migration(s) in the reverse direction. If the "version" is left
	unspecified, then only the first available migration is executed. Otherwise,
	available migrations are executed down to and including the specified
	version. Specify a version in the form: %s.`,
				bin, godfish.TimeFormat)
			return
		},
		mapper: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			if a.Version == "" {
				err = godfish.ApplyMigration(
					driver,
					pathToDBMigrations,
					godfish.DirReverse,
					a.Version,
				)
			} else {
				err = godfish.Migrate(
					driver,
					pathToDBMigrations,
					godfish.DirReverse,
					a.Version,
				)
			}
			return err
		},
	},
}

// pathToDBMigrations is the path, relative from the project root, to the
// database migrations directory. This will only work if you are running this
// command from the project root as well.
const pathToDBMigrations = "db/migrations"

var (
	args Args
	bin  = os.Args[0]
)

func init() {
	flag.Usage = func() {
		cmds := make([]string, 0)
		for cmd := range commands {
			cmds = append(
				cmds,
				fmt.Sprintf("%-20s\t%-40s", cmd, commands[cmd].description),
			)
		}
		sort.Strings(cmds)
		fmt.Fprintf(
			flag.CommandLine.Output(), `Usage: %s -cmd command [arguments]

Commands:

%v`, os.Args[0], strings.Join(cmds, "\n"),
		)
		fmt.Printf("\n\nFlags:\n\n")
		flag.PrintDefaults()
	}

	flag.StringVar(&args.Cmd, "cmd", "", "name of command to execute")
	flag.StringVar(&args.Name, "name", "", "create a migration with a name, ie: foo")
	flag.StringVar(
		&args.Version,
		"version",
		"",
		fmt.Sprintf("timestamp of migration, format: %s", godfish.TimeFormat),
	)
	flag.StringVar(
		&args.Direction,
		"direction",
		"forward",
		"direction of migration to run. [forward | reverse]",
	)
	flag.StringVar(&args.Driver, "driver", "", "name of database driver, ie: postgres, mysql")
	flag.BoolVar(&args.Reversible, "reversible", true, "create a reversible migration?")
	flag.BoolVar(&args.Help, "h", false, "show help menu")
	flag.BoolVar(&args.Help, "help", false, "show help menu")
	flag.Parse()
}

func main() {
	var err error
	if cmd, ok := commands[args.Cmd]; !ok {
		flag.Usage()
		err = fmt.Errorf("unknown command %q", args.Cmd)
	} else if args.Help {
		fmt.Fprint(
			flag.CommandLine.Output(),
			cmd.help(),
		)
		fmt.Println()
	} else {
		err = cmd.mapper(args)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newDriver(driverName string) (driver godfish.Driver, err error) {
	switch driverName {
	case "postgres":
		driver, err = godfish.NewDriver(driverName, godfish.PostgresParams{
			Encoding: "UTF8",
			Host:     "localhost",
			Name:     os.Getenv("DB_NAME"),
			Pass:     os.Getenv("DB_PASSWORD"),
			Port:     "5432",
		})
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
