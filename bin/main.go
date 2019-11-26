package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"bitbucket.org/rafaelespinoza/godfish"
)

// Args describes the CLI inputs.
type Args struct {
	Cmd        string
	Help       bool
	Name       string
	Reversible bool
	Version    string
	Direction  string
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
	"generate-migration": command{
		description: "create migration files",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: generate-migration -name <name> [-rev]

	Creates migration files at %s: one meant for the "forward" direction,
	another meant for "reverse". Optionally create a migration in the forward
	direction only by passing the flag "-rev=false". The "name" flag has no
	effects other than on the generated filename. The output filename
	automatically has a "version", which is a timestamp in the form: %s.`,
				pathToDBMigrations, godfish.TimeFormat)
			return
		},
		mapper: func(a Args) error {
			return generateMigration(a.Name, a.Reversible)
		},
	},
	"run-all-migrations": command{
		description: "execute many migration files at once",
		help: func() (out string) {
			out = `
Usage: run-all-migrations -dir [forward|reverse]

	Execute several migration files in either forward or reverse directions.
	Specify the direction using the "dir" flag.`
			return
		},
		mapper: func(a Args) error {
			direction := godfish.DirForward
			if strings.HasPrefix(a.Direction, "rev") || strings.HasPrefix(a.Direction, "back") {
				direction = godfish.DirReverse
			}
			return runAllMigrations(direction, "", "")
		},
	},
	"run-migration": command{
		description: "execute migration files",
		help: func() (out string) {
			out = fmt.Sprintf(`
Usage: run-migration -dir [forward|reverse] -version <timestamp>

	Execute migration files. Specify the migration direction with the "dir"
	flag, which is "forward" by default. The "version" flag is used to specify
	which migration to run. It should be a timestamp in the form: %s.`,
				godfish.TimeFormat)
			return
		},
		mapper: func(a Args) error {
			direction := godfish.DirForward
			if strings.HasPrefix(a.Direction, "rev") || strings.HasPrefix(a.Direction, "back") {
				direction = godfish.DirReverse
			}
			return runMigration(a.Version, direction)
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
			flag.CommandLine.Output(), `Usage: %s <command> [arguments]

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

func generateMigration(name string, reversible bool) error {
	dir, err := os.Open(pathToDBMigrations)
	if err != nil {
		return err
	}
	migration, err := godfish.NewMigration(name, reversible, dir)
	if err != nil {
		return err
	}
	return migration.GenerateFiles()
}

func runAllMigrations(direction godfish.Direction, versionLo, versionHi string) (err error) {
	// get all filenames in db migration path that match direction, sort them.
	var fileDir *os.File
	var filenames []string
	var mutations []godfish.Mutation
	var dbHandler *sql.DB

	if fileDir, err = os.Open(pathToDBMigrations); err != nil {
		return
	}
	defer fileDir.Close()
	if filenames, err = fileDir.Readdirnames(0); err != nil {
		return
	}
	fmt.Printf("filenames: %#v\n", filenames)
	for _, filename := range filenames {
		var mut godfish.Mutation
		if mut, err = godfish.ParseMutation(godfish.Filename(filename)); err != nil {
			return
		}
		if mut.Direction() != direction {
			continue
		}
		// TODO: make use of the versionLo, versionHi parameters to only run
		// migrations in a selected datetime range.
		mutations = append(mutations, mut)
	}
	if dbHandler, err = connectToDB(); err != nil {
		return
	}
	for _, mut := range mutations {
		var relPath string
		if relPath, err = godfish.PathToMutationFile(pathToDBMigrations, mut); err != nil {
			return
		}
		if err = godfish.RunMutation(dbHandler, relPath); err != nil {
			return
		}
	}

	return
}

func runMigration(version string, direction godfish.Direction) (err error) {
	var baseGlob godfish.Filename
	var filenames []string
	var mut godfish.Mutation
	var dbHandler *sql.DB
	var pathToMigrationFile string

	if direction == godfish.DirUnknown {
		err = fmt.Errorf("unknown direction")
		return
	}

	if baseGlob, err = godfish.MakeFilename(version, direction, "*"); err != nil {
		return
	}
	if filenames, err = filepath.Glob(pathToDBMigrations + "/" + string(baseGlob)); err != nil {
		return
	} else if len(filenames) == 0 {
		err = fmt.Errorf("could not find matching files")
		return
	} else if len(filenames) > 1 {
		err = fmt.Errorf("need 1 matching filename; got %v", filenames)
		return
	}
	if mut, err = godfish.ParseMutation(godfish.Filename(filenames[0])); err != nil {
		return
	}
	if dbHandler, err = connectToDB(); err != nil {
		return
	}
	if pathToMigrationFile, err = godfish.PathToMutationFile(pathToDBMigrations, mut); err != nil {
		return
	}
	return godfish.RunMutation(dbHandler, pathToMigrationFile)
}

func connectToDB() (dbHandler *sql.DB, err error) {
	dbHandler, err = godfish.Connect(
		"postgres",
		godfish.PGParams{
			Encoding: "UTF8",
			Host:     "localhost",
			Name:     os.Getenv("DB_NAME"),
			Pass:     os.Getenv("DB_PASSWORD"),
			Port:     "5432",
		},
	)
	return
}
