// Package commands contains all the CLI stuff.
package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/rafaelespinoza/godfish"
)

// arguments describes the CLI inputs and other configuration variables.
type arguments struct {
	Conf       string
	Debug      bool
	Direction  string
	DSN        godfish.DSN
	Files      string
	Name       string
	Reversible bool
	Version    string
}

var (
	// args is the shared set of named values.
	args arguments
	// bin is the name of the binary.
	bin = os.Args[0]
)

// Run does all the CLI things.
func Run(dsn godfish.DSN) (err error) {
	flag.Parse()
	args.DSN = dsn

	var cmd *subcommand

	if cmd, err = initSubcommand(flag.Args(), &args); cmd == nil {
		// either asked for help or asked for unknown command
		flag.Usage()
	}
	if err != nil {
		return
	}

	err = cmd.run(args)
	return
}

func init() {
	flag.Usage = func() {
		cmds := make([]string, 0)
		for cmd := range subcommands {
			cmds = append(
				cmds,
				fmt.Sprintf("%-20s\t%-40s", cmd, subcommands[cmd].description),
			)
		}
		sort.Strings(cmds)
		fmt.Fprintf(flag.CommandLine.Output(), `Usage:

	%s [flags] command [sub-flags]

Description:

	godfish is a database migration manager. It tracks the status of migrations
	by recording a timestamp in a table called "schema_migrations" in the
	"migration_id" column. Those timestamps correspond to SQL migration files
	that you write and store somewhere on the filesystem. You need to configure
	the path to the SQL migration files as well as the name of the driver to use
	(ie: postgres, mysql, potato, potato).

	Configuration options are set with flags or with a configuration file. Options
	specified via flags will take precedence over the config file.

	Specify database connection params with environment variable:
		DB_DSN=

	The following flags should go before the command.`,
			bin)
		printFlagDefaults(flag.CommandLine)
		fmt.Fprintf(
			flag.CommandLine.Output(), `
Commands:

	These will have their own set of flags. Put them after the command.

	%v

Examples:

	%s [command] -h
`,
			strings.Join(cmds, "\n\t"), bin)

	}

	flag.StringVar(&args.Conf, "conf", ".godfish.json", "path to godfish config file")
	flag.BoolVar(&args.Debug, "debug", false, "output extra debugging info")
	flag.StringVar(
		&args.Files,
		"files",
		"",
		"path to migration files, can also set with config file",
	)
}

// initSubcommand selects the sub command to run. If the command name is not
// found, then it outputs help. If the command is found, then merge config file
// with CLI args and set up the Subcommand.
func initSubcommand(positionalArgs []string, a *arguments) (subcmd *subcommand, err error) {
	if len(positionalArgs) == 0 || positionalArgs[0] == "help" {
		err = flag.ErrHelp
		return
	} else if c, ok := subcommands[positionalArgs[0]]; !ok {
		err = fmt.Errorf("unknown command %q", positionalArgs[0])
		return
	} else {
		subcmd = c
	}

	// Read configuration file, if present. Negotiate with Args.
	var conf godfish.MigrationsConf
	if data, ierr := os.ReadFile(a.Conf); ierr != nil {
		// probably no config file present, rely on Args instead.
	} else if ierr = json.Unmarshal(data, &conf); ierr != nil {
		err = ierr
		return
	}
	if a.Files == "" && conf.PathToFiles != "" {
		a.Files = conf.PathToFiles
	}

	if a.Debug {
		fmt.Printf("positional arguments: %#v\n", flag.Args())
		fmt.Printf("config file at %q: %#v\n", a.Conf, conf)
		fmt.Printf("Args prior subcmd flag parse: %#v\n", a)
	}
	subflags := subcmd.setup(a)
	if err = subflags.Parse(positionalArgs[1:]); err != nil {
		return
	}
	if a.Debug {
		fmt.Printf("Args after subcmd flag parse: %#v\n", a)
	}
	return
}

type subcommand struct {
	// description should provide a short summary.
	description string
	// setup should prepare Args for interpretation by using the pointer to Args
	// with the returned flag set.
	setup func(a *arguments) *flag.FlagSet
	// run is a wrapper function that selects the necessary command line inputs,
	// executes the command and returns any errors.
	run func(a arguments) error
}

// subcommands registers any operation by name to a subcommand.
var subcommands = map[string]*subcommand{
	"create-migration": &subcommand{
		description: "generate migration files",
		setup: func(a *arguments) *flag.FlagSet {
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
		run: func(a arguments) error {
			migration, err := godfish.NewMigrationParams(a.Name, a.Reversible, a.Files)
			if err != nil {
				return err
			}
			return migration.GenerateFiles()
		},
	},
	"dump-schema": &subcommand{
		description: "generate a sql file describing the db schema",
		setup: func(a *arguments) *flag.FlagSet {
			flags := flag.NewFlagSet("dump-schema", flag.ExitOnError)
			flags.Usage = func() {
				fmt.Printf(`Usage: %s dump-schema

	Print a database structure file to standard output.`,
					bin)
				printFlagDefaults(flags)
			}
			return flags
		},
		run: func(a arguments) error {
			driver, err := bootDriver(a.DSN)
			if err != nil {
				return err
			}
			godfish.DumpSchema(driver)
			return nil
		},
	},
	"info": &subcommand{
		description: "output current state of migrations",
		setup: func(a *arguments) *flag.FlagSet {
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
		run: func(a arguments) error {
			driver, err := bootDriver(a.DSN)
			if err != nil {
				return err
			}
			direction := whichDirection(a)
			return godfish.Info(driver, a.Files, direction, a.Version)
		},
	},
	"init": &subcommand{
		description: "create godfish configuration file",
		setup: func(a *arguments) *flag.FlagSet {
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
		run: func(a arguments) error {
			return godfish.Init(a.Conf)
		},
	},
	"migrate": &subcommand{
		description: "execute migration(s) in the forward direction",
		setup: func(a *arguments) *flag.FlagSet {
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
		run: func(a arguments) error {
			driver, err := bootDriver(a.DSN)
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
	"remigrate": &subcommand{
		description: "rollback and then re-apply the last migration",
		setup: func(a *arguments) *flag.FlagSet {
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
		run: func(a arguments) error {
			driver, err := bootDriver(a.DSN)
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
	"rollback": &subcommand{
		description: "execute migration(s) in the reverse direction",
		setup: func(a *arguments) *flag.FlagSet {
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
		run: func(a arguments) error {
			driver, err := bootDriver(a.DSN)
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
	"version": _Version,
}

func bootDriver(dsn godfish.DSN) (driver godfish.Driver, err error) {
	connParams := godfish.ConnectionParams{
		Host: os.Getenv("DB_HOST"),
		Name: os.Getenv("DB_NAME"),
		Pass: os.Getenv("DB_PASSWORD"),
		Port: os.Getenv("DB_PORT"),
		User: os.Getenv("DB_USER"),
	}
	if err = dsn.Boot(connParams); err != nil {
		return
	}
	driver, err = dsn.NewDriver(nil)
	return
}

func whichDirection(a arguments) (direction godfish.Direction) {
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
