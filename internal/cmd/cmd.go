// Package cmd contains all the CLI stuff.
package cmd

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
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
	theDriver DriverConnector
)

// Root abstracts a top-level command from package main.
type Root interface {
	// Run is the entry point. It should be called with os.Args[1:].
	Run(ctx context.Context, args []string) error
}

// New constructs a top-level command with subcommands.
func New(driver DriverConnector, sampleDSN string) Root {
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
	by recording a timestamp in a table, by default called %q,
	in the "migration_id" column. Those timestamps correspond to SQL migration
	files that you write and store somewhere on the filesystem. You need to
	configure the path to the SQL migration files as well as the name of the driver
	to use (ie: postgres, mysql, potato, potato).

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
			bin, internal.DefaultMigrationsTableName, internal.DSNKey, sampleDSN)
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
	rootFlags.StringVar(
		&commonArgs.MigrationsTable,
		"migrations-table",
		internal.DefaultMigrationsTableName,
		"name of DB table for storing migration state",
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

			pathToConfig = filepath.Clean(pathToConfig)
			slog.Debug("cmd: before loading config file", slog.String("path_to_config", pathToConfig), slog.Any("common_args", commonArgs))

			// Look for config file and if present, merge those values with
			// input flag values.
			dir := filepath.Dir(pathToConfig)
			conf, err := loadConfig(os.DirFS(dir), filepath.Base(pathToConfig))
			if err != nil {
				return err
			}
			commonArgs = resolveConfig(rootFlags, commonArgs, conf)

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
	MigrationsTable                  string
}

// LogValue lets this type implement the [slog.LogValuer] interface.
func (c commonArguments) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("files", c.Files),
		slog.String("data_source_name", c.DataSourceName),
		slog.String("default_fwd_label", c.DefaultFwdLabel),
		slog.String("default_rev_label", c.DefaultRevLabel),
		slog.String("migrations_table", c.MigrationsTable),
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

var errReadConfig = errors.New("reading config file")

// loadConfig reads a configuration file and parses its contents. If the file is
// not found, then it returns a zero value configuration without any error.
func loadConfig(fsys fs.FS, basename string) (conf internal.Config, err error) {
	var data []byte
	if data, err = fs.ReadFile(fsys, basename); errors.Is(err, fs.ErrNotExist) {
		// Probably no config file present. That's ok.
		slog.Debug("cmd: attempted to read config file, relying on arguments instead", slog.Any("error", err), slog.String("filename", basename))
		err = nil // Zero everything out and have the caller continue on.
		return
	} else if err != nil {
		err = fmt.Errorf("%w %s: %w", errReadConfig, basename, err)
		return
	}

	if err = json.Unmarshal(data, &conf); err != nil {
		err = fmt.Errorf("%w, parsing config data from file %q: %w", internal.ErrDataInvalid, basename, err)
	}
	return
}

// resolveConfig reconciles configuration values, preferring values passed by
// CLI flag over values set in the configuration file.
func resolveConfig(flags *flag.FlagSet, curr commonArguments, conf internal.Config) (next commonArguments) {
	next = curr

	next.Files = resolveConfigVal(flags, "files", conf.PathToFiles, curr.Files)
	next.MigrationsTable = resolveConfigVal(flags, "migrations-table", conf.MigrationsTable, internal.DefaultMigrationsTableName)

	// Subcommands may override these with their own flags.
	next.DefaultFwdLabel = cmp.Or(next.DefaultFwdLabel, conf.ForwardLabel)
	next.DefaultRevLabel = cmp.Or(next.DefaultRevLabel, conf.ReverseLabel)
	return
}

// resolveConfigVal sets a configuration value with this priority (lowest to highest):
//
//  1. defaultVal (lowest priority)
//  2. valFromConf
//  3. value from flag (highest priority)
func resolveConfigVal(flags *flag.FlagSet, targetFlagName, valFromConf, defaultVal string) string {
	out := defaultVal

	if valFromConf != "" {
		out = valFromConf
	}

	// Was this flag set? If so, use the corresponding value.
	flags.Visit(func(f *flag.Flag) {
		if f.Name == targetFlagName {
			out = f.Value.String()
		}
	})

	return out
}

var exampleDurationVals = []string{"30s", "5m", "1h2m3s"}

// DriverConnector is a godfish Driver with connection management.
type DriverConnector interface {
	godfish.Driver
	Connector
}

// Connector manages DB connections.
type Connector interface {
	// Connect should open a connection to the database.
	Connect(dsn string) error
	// Close should close the database connection.
	Close() error
}

// withConnection runs a DB operation f after connecting to the DB and before
// closing the connection. The callback function f is passed the context ctx,
// and is meant as a placeholder for a godfish.Driver.
//
// The input dsn (data source name) is a DB-specific connection string, and is
// a soft requirement. If empty then it looks up an environment variable DB_DSN.
// In the case that the input dsn is empty and the env var is unset or empty,
// then this func returns with an error.
func withConnection(ctx context.Context, dsn string, conn Connector, f func(context.Context) error) (err error) {
	if dsn == "" {
		if dsn, err = getDSN(); err != nil {
			err = fmt.Errorf("missing input dsn, %w", err)
			return
		}
	}

	if err = conn.Connect(dsn); err != nil {
		return err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			slog.Warn("closing driver", slog.Any("error", cerr))
		}
	}()

	err = f(ctx)
	return
}

func getDSN() (dsn string, err error) {
	dsn = os.Getenv(internal.DSNKey)
	if dsn == "" {
		err = fmt.Errorf("missing environment variable: %s", internal.DSNKey)
	}
	return
}
