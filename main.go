package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"bitbucket.org/rafaelespinoza/godfish/godfish"
)

// Args describes the CLI inputs and other configuration variables.
type Args struct {
	Conf       string
	Debug      bool
	Direction  string
	Driver     string
	Files      string
	Name       string
	Reversible bool
	Version    string
}

var (
	// args is the shared set of named values.
	args Args
	// bin is the name of the binary.
	bin = os.Args[0]
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

	Specify database connection params with environment variables:
		DB_HOST=
		DB_NAME=
		DB_PASSWORD=
		DB_PORT=
		DB_USER=

	The following flags should go before the command.`,
			bin)
		printFlagDefaults(flag.CommandLine)
		fmt.Fprintf(
			flag.CommandLine.Output(), `
Commands:

	These will have their own set of flags. Put them after the command.

	%v

Examples:

	%s -h
	%s [command] -h
`,
			strings.Join(cmds, "\n\t"), bin, bin)

	}

	flag.StringVar(&args.Conf, "conf", ".godfish.json", "path to godfish config file")
	flag.BoolVar(&args.Debug, "debug", false, "output extra debugging info")
	flag.StringVar(
		&args.Driver,
		"driver",
		"",
		"name of driver, can also set with config file",
	)
	flag.StringVar(
		&args.Files,
		"files",
		"",
		"path to migration files, can also set with config file",
	)
}

// initCommand selects the sub command to run. If the command name is not found,
// then it outputs help. If the command is found, then merge config file with
// CLI args and set up the command.
func initCommand(positionalArgs []string, a *Args) (subcmd *command, err error) {
	if len(positionalArgs) == 0 || positionalArgs[0] == "help" {
		err = flag.ErrHelp
		return
	} else if c, ok := commands[positionalArgs[0]]; !ok {
		err = fmt.Errorf("unknown command %q", positionalArgs[0])
		return
	} else {
		subcmd = c
	}

	// Read configuration file, if present. Negotiate with Args.
	var conf godfish.MigrationsConf
	if data, ierr := ioutil.ReadFile(a.Conf); ierr != nil {
		// probably no config file present, rely on Args instead.
	} else if ierr = json.Unmarshal(data, &conf); ierr != nil {
		err = ierr
		return
	}
	if a.Driver == "" && conf.DriverName != "" {
		a.Driver = conf.DriverName
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

func main() {
	flag.Parse()

	var cmd *command
	var err error

	if cmd, err = initCommand(flag.Args(), &args); cmd == nil {
		// either asked for help or asked for unknown command
		flag.Usage()
		fmt.Println(err)
		os.Exit(1)
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = cmd.run(args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
