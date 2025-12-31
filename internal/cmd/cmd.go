// Package cmd contains all the CLI stuff.
package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafaelespinoza/alf"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

var (
	commonArgs commonArguments
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
func New(driver godfish.Driver, sampleDSN string) Root {
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
		_, _ = fmt.Fprintf(rootFlags.Output(), `Usage:

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

	Database connection params can be specified in these ways:
		* Environment variable: %s
		* Command line flag, -dsn. This has higher precedence than the
		  environment variable.
	Sample DSN:
		%s

	The following flags should go before the command.
`,
			bin, internal.DSNKey, sampleDSN)
		printFlagDefaults(rootFlags)
		_, _ = fmt.Fprintf(
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
	rootFlags.StringVar(
		&commonArgs.DataSourceName,
		"dsn",
		"",
		fmt.Sprintf("database DSN, if empty then fallback to environment variable %s", internal.DSNKey),
	)
	rootFlags.BoolVar(&loggingOff, "q", false, "if true, then all logging is effectively off")
	rootFlags.StringVar(&logLevel, "loglevel", defaultLoggingLevel.String(), fmt.Sprintf("minimum severity for which to log events, should be one of %q", validLoggingLevels))
	rootFlags.StringVar(&logFormat, "logformat", defaultLoggingFormat, fmt.Sprintf("output format for logs, should be one of %q", validLoggingFormats))
	del.Flags = rootFlags

	return &alf.Root{
		Delegator: del,
		PrePerform: func(_ context.Context) error {
			handler := newLogHandler(os.Stderr, loggingOff, logLevel, logFormat)
			slog.SetDefault(slog.New(handler))

			slog.Debug("cmd: before loading config file", slog.String("path_to_config", pathToConfig), slog.Any("common_args", commonArgs))

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

			slog.Debug("cmd: after loading config file", slog.Any("conf", conf))

			if val := strings.TrimSpace(commonArgs.DataSourceName); val != "" {
				err := os.Setenv(internal.DSNKey, val)
				if err != nil {
					return fmt.Errorf("setting env var %s with flag -dsn: %w", internal.DSNKey, err)
				}
			}

			slog.Debug("cmd: after resolving config values", slog.Any("common_args", commonArgs))
			return nil
		},
	}
}

// commonArguments are read from a configuration file, if available. The
// subcommand code is written so that flag values may take precedence over
// values in here.
type commonArguments struct {
	Files                            string
	DataSourceName                   string
	DefaultFwdLabel, DefaultRevLabel string
}

// LogValue lets this type implement the [slog.LogValuer] interface.
func (c commonArguments) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("files", c.Files),
		slog.String("data_source_name", c.DataSourceName),
		slog.String("default_fwd_label", c.DefaultFwdLabel),
		slog.String("default_rev_label", c.DefaultRevLabel),
	)
}

func newFlagSet(name string) (out *flag.FlagSet) {
	out = flag.NewFlagSet(name, flag.ContinueOnError)
	out.SetOutput(os.Stdout)
	return
}

// printFlagDefaults calls PrintDefaults on f. It helps make help message
// formatting more consistent.
func printFlagDefaults(f *flag.FlagSet) {
	_, _ = fmt.Fprintf(f.Output(), "\n%s flags:\n\n", f.Name())
	f.PrintDefaults()
}
