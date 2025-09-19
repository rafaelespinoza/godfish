// Package cmd contains all the CLI stuff.
package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafaelespinoza/alf"
	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/log"
)

var (
	// commonArgs values are read from a configuration file, if available. The
	// subcommand code is written so that flag values may take precedence over
	// values in here.
	commonArgs struct {
		Files                            string
		DefaultFwdLabel, DefaultRevLabel string
	}
	// bin is the name of the binary.
	bin = os.Args[0]
	// theDriver is passed in from a Driver's package main.
	theDriver godfish.Driver
)

// Root abstracts a top-level command from package main.
type Root interface {
	// Run is the entry point. It should be called with os.Args[1:].
	Run(ctx context.Context, args []string) error
}

// New constructs a top-level command with subcommands.
func New(driver godfish.Driver) Root {
	theDriver = driver
	del := &alf.Delegator{
		Description: "main command for " + bin,
		Subs: map[string]alf.Directive{
			"create-migration": makeCreateMigration("create-migration"),
			"info":             makeInfo("info"),
			"init":             makeInit("init"),
			"migrate":          makeMigrate("migrate"),
			"remigrate":        makeRemigrate("remigrate"),
			"rollback":         makeRollback("rollback"),
			"version":          makeVersion("version"),
		},
	}

	rootFlags := newFlagSet("godfish")
	rootFlags.Usage = func() {
		fmt.Fprintf(rootFlags.Output(), `Usage:

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

	The following flags should go before the command.
`,
			bin)
		printFlagDefaults(rootFlags)
		fmt.Fprintf(
			rootFlags.Output(), `
Commands:

	These will have their own set of flags. Put them after the command.

	%v

Examples:

	%s [command] -h
`,
			strings.Join(del.DescribeSubcommands(), "\n\t"), bin)
	}

	var (
		pathToConfig string

		// Other args for logging. For now the inputs are flags, but maybe put
		// into configuration file as well.
		loggingOff bool
		logLevel   string
		logFormat  string
	)

	rootFlags.StringVar(&pathToConfig, "conf", ".godfish.json", "path to godfish config file")
	rootFlags.StringVar(
		&commonArgs.Files,
		"files",
		"",
		"path to migration files, can also set with config file",
	)
	rootFlags.BoolVar(&loggingOff, "q", false, "if true, then all logging is effectively off")
	rootFlags.StringVar(&logLevel, "loglevel", log.Levels[0].String(), fmt.Sprintf("minimum severity for which to log events, should be one of %q", log.Levels))
	rootFlags.StringVar(&logFormat, "logformat", log.Formats[len(log.Formats)-1], fmt.Sprintf("output format for logs, should be one of %q", log.Formats))
	del.Flags = rootFlags

	return &alf.Root{
		Delegator: del,
		PrePerform: func(_ context.Context) error {
			var w io.Writer
			if !loggingOff {
				w = os.Stderr
			}
			log.SetLogger(w, logLevel, logFormat)

			// Look for config file and if present, merge those values with
			// input flag values.
			var conf internal.Config
			if data, ierr := os.ReadFile(filepath.Clean(pathToConfig)); ierr != nil {
				// probably no config file present, rely on arguments instead.
			} else if ierr = json.Unmarshal(data, &conf); ierr != nil {
				return ierr
			}
			if commonArgs.Files == "" && conf.PathToFiles != "" {
				commonArgs.Files = conf.PathToFiles
			}

			// Subcommands may override these with their own flags.
			commonArgs.DefaultFwdLabel = conf.ForwardLabel
			commonArgs.DefaultRevLabel = conf.ReverseLabel

			return nil
		},
	}
}

func newFlagSet(name string) (out *flag.FlagSet) {
	out = flag.NewFlagSet(name, flag.ContinueOnError)
	out.SetOutput(os.Stdout)
	return
}

// printFlagDefaults calls PrintDefaults on f. It helps make help message
// formatting more consistent.
func printFlagDefaults(f *flag.FlagSet) {
	fmt.Fprintf(f.Output(), "\n%s flags:\n\n", f.Name())
	f.PrintDefaults()
}
