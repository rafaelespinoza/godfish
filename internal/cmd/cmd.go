// Package cmd contains all the CLI stuff.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"

	altsrc "github.com/urfave/cli-altsrc/v3"
	jsonsrc "github.com/urfave/cli-altsrc/v3/json"
	"github.com/urfave/cli/v3"
)

// Root abstracts a top-level command from package main.
type Root interface {
	// Run is the entry point. It should be called with os.Args.
	Run(ctx context.Context, args []string) error
}

// New constructs a top-level command with subcommands.
func New(driver DriverConnector, sampleDSN string) Root {
	const defaultConfigFilepath = ".godfish.json"
	pathToConfig := defaultConfigFilepath

	cmd := &cli.Command{
		Name:  filepath.Base(os.Args[0]),
		Usage: fmt.Sprintf("Manage %s DB migrations", driver.Name()),
		Description: fmt.Sprintf(`godfish is a database migration manager. It tracks the status of migrations
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
	%s`,
			internal.DefaultMigrationsTableName, internal.DSNKey, sampleDSN,
		),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "conf",
				Destination: &pathToConfig,
				Value:       defaultConfigFilepath,
				Usage:       "path to godfish config file",
				TakesFile:   true,
			},
			&cli.StringFlag{
				Name:      pathToFilesFlagname,
				Usage:     "path to migration files, can also set with config file",
				TakesFile: true,
				Sources:   newSourceConfigChain(&pathToConfig, "path_to_files"),
			},
			&cli.StringFlag{
				Name:    "dsn",
				Usage:   fmt.Sprintf("database DSN, if empty then fallback to environment variable %s", internal.DSNKey),
				Sources: newSourceConfigChain(&pathToConfig, "dsn"),
			},
			&cli.StringFlag{
				Name:    migrationsTableFlagname,
				Value:   internal.DefaultMigrationsTableName,
				Usage:   "name of DB table for storing migration state",
				Sources: newSourceConfigChain(&pathToConfig, "migrations_table"),
			},
			&cli.BoolFlag{
				Name:  "q",
				Usage: "if true, then all logging is effectively off",
			},
			&cli.StringFlag{
				Name:  "loglevel",
				Value: defaultLoggingLevel.String(),
				Usage: fmt.Sprintf("minimum severity for which to log events, should be one of %q", validLoggingLevels),
			},
			&cli.StringFlag{
				Name:  "logformat",
				Value: defaultLoggingFormat,
				Usage: fmt.Sprintf("output format for logs, should be one of %q", validLoggingFormats),
			},
		},
		Commands: []*cli.Command{
			makeApplyMigration("apply-migration"),
			makeCreateMigration("create-migration", &pathToConfig),
			makeInfo("info"),
			makeInit("init"),
			makeMigrate("migrate"),
			makeRemigrate("remigrate"),
			makeRollback("rollback"),
			makeUpgradeSchemaMigrations(upgradeCmdName, &pathToConfig),
			makeVersion("version"),
		},
		CommandNotFound: func(ctx context.Context, c *cli.Command, input string) {
			if err := renderCommandNotFound(c, input, c.Writer); err != nil {
				slog.Error("attempting to render not found message", slog.Any("error", err.Error()))
			}
			cli.HandleExitCoder(cli.Exit("subcommand not found", 2))
		},
		Version: versionTag,
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			ctx = setDriver(ctx, driver)

			handler := newLogHandler(os.Stderr, c.Bool("q"), c.String("loglevel"), c.String("logformat"))
			slog.SetDefault(slog.New(handler))

			if val := strings.TrimSpace(c.String("dsn")); val != "" {
				err := os.Setenv(internal.DSNKey, val)
				if err != nil {
					return ctx, fmt.Errorf("setting env var %s with flag -dsn: %w", internal.DSNKey, err)
				}
			}

			slog.Debug("input args before running command",
				slog.Any("args", c.Args().Slice()),
				slog.Int("num_flags_set", c.NumFlags()),
				slog.GroupAttrs("flags",
					slog.String("conf", c.String("conf")),
					slog.String(pathToFilesFlagname, c.String(pathToFilesFlagname)),
					slog.String("dsn", c.String("dsn")),
					slog.String(migrationsTableFlagname, c.String(migrationsTableFlagname)),
					slog.Bool("q", c.Bool("q")),
					slog.String("loglevel", c.String("loglevel")),
					slog.String("logformat", c.String("logformat")),
				),
			)

			return ctx, nil
		},
		EnableShellCompletion: true,
		Suggest:               true,
	}

	return cmd
}

// Keep references to names of flags consistent.
const (
	pathToFilesFlagname     = "files"
	migrationsTableFlagname = "migrations-table"
)

// newSourceConfigChain is for use on flags that may have values set from a configuration file.
func newSourceConfigChain(pathToConfigFile *string, key string) cli.ValueSourceChain {
	return cli.NewValueSourceChain(
		jsonsrc.JSON(key, altsrc.NewStringPtrSourcer(pathToConfigFile)),
	)
}

var exampleDurationVals = []string{"30s", "5m", "1h2m3s"}

// DriverConnector is a godfish Driver with connection management.
type DriverConnector interface {
	godfish.Driver
	Connector
}

type driverCtxKey struct{}

// setDriver puts dc on ctx. Use [getDriver] to retrieve it.
func setDriver(ctx context.Context, dc DriverConnector) context.Context {
	return context.WithValue(ctx, driverCtxKey{}, dc)
}

// getDriver retrieves a DriverConnector from the context and nil when
// a DriverConnector value was previously placed there using [setDriver].
// If the context does not have this value, then it returns nil and an error
// reminding you to properly set up the driver.
func getDriver(ctx context.Context) (DriverConnector, error) {
	dc, found := ctx.Value(driverCtxKey{}).(DriverConnector)
	if !found {
		return nil, errNoDriver
	}
	return dc, nil
}

var errNoDriver = errors.New("driver uninitialized")

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

const commandNotFoundTemplate = `Subcommand "{{.Input}}" not found.

Available commands:
{{- range $subcmd := .VisibleCommands}}
  {{$subcmd.Name}}
{{- end}}

{{with .Suggestion}}Did you mean "{{.}}"?{{end}}
`

func renderCommandNotFound(c *cli.Command, input string, w io.Writer) error {
	tmpl, err := template.New("command not found").Parse(commandNotFoundTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	var data = struct {
		Input           string
		VisibleCommands []*cli.Command
		Suggestion      string
	}{
		Input:           input,
		VisibleCommands: c.VisibleCommands(),
		Suggestion:      cli.SuggestCommand(c.Commands, input),
	}

	if err = tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	return nil
}
