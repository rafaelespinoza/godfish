package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"bitbucket.org/rafaelespinoza/godfish"
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
	"dump-schema": command{
		description: "generate a sql file describing the db schema",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: dump-schema -driver <driverName>
	`)
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
	"generate-migration": command{
		description: "create migration files",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: generate-migration -name <name> [-reversible]

	Creates migration files at %s: one meant for the "forward" direction,
	another meant for "reverse". Optionally create a migration in the forward
	direction only by passing the flag "-reversible=false". The "name" flag has
	no effects other than on the generated filename. The output filename
	automatically has a "version", which is a timestamp in the form: %s.`,
				pathToDBMigrations, godfish.TimeFormat)
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
	"info": command{
		description: "output current state of migrations",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: info -direction [forward|reverse] -driver <driverName>
			`)
			return
		},
		mapper: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			direction := whichDirection(a)
			return godfish.Info(driver, direction, pathToDBMigrations)
		},
	},
	"init": command{
		description: "creates schema migrations table",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: init -driver <driverName>

Creates the db table to track migrations, unless it already exists.`)
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
	"run-all": command{
		description: "execute all migration files",
		help: func() (out string) {
			out = `
Usage: run-all -direction [forward|reverse] -driver <driverName>

	Execute all migration files in either forward or reverse directions.
	Specify the direction using the "direction" flag.`
			return
		},
		mapper: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			direction := whichDirection(a)
			return godfish.Migrate(driver, direction, pathToDBMigrations)
		},
	},
	"run-one": command{
		description: "execute one migration",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: run-one -direction [forward|reverse] -driver <driverName> -version <timestamp>

	Execute just one migration file. Specify the migration direction with the
	"direction" flag, which is "forward" by default. The "version" flag is used
	to specify which migration to run. It should be a timestamp in the form:
	%s.`,
				godfish.TimeFormat)
			return
		},
		mapper: func(a Args) error {
			driver, err := newDriver(a.Driver)
			if err != nil {
				return err
			}
			direction := whichDirection(a)
			return godfish.ApplyMigration(
				driver,
				direction,
				pathToDBMigrations,
				args.Version,
			)
		},
	},
}

// pathToDBMigrations is the path, relative from the project root, to the
// database migrations directory. This will only work if you are running this
// command from the project root as well.
const pathToDBMigrations = "db/migrations"

var args Args

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
	flag.StringVar(&args.Name, "name", "", "generate a migration with a name, ie: foo")
	flag.StringVar(
		&args.Version,
		"version",
		"",
		fmt.Sprintf("timestamp of migration to run, format: %s", godfish.TimeFormat),
	)
	flag.StringVar(
		&args.Direction,
		"direction",
		"forward",
		"direction of migration to run. [forward | reverse]",
	)
	flag.StringVar(&args.Driver, "driver", "", "name of database driver, ie: postgres, mysql")
	flag.BoolVar(&args.Reversible, "reversible", true, "generate a reversible migration?")
	flag.BoolVar(&args.Help, "h", false, "show help menu")
	flag.BoolVar(&args.Help, "help", false, "show help menu")
	flag.Parse()
}

func main() {
	var err error
	if cmd, ok := commands[args.Cmd]; !ok {
		flag.Usage()
		err = fmt.Errorf("unknown command %q\n", args.Cmd)
	} else if args.Help {
		flag.Usage()
		fmt.Fprint(
			flag.CommandLine.Output(),
			cmd.help(),
		)
		fmt.Println()
	} else {
		err = cmd.mapper(args)
	}
	if err != nil {
		panic(err)
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
