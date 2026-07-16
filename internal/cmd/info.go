package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"

	"github.com/urfave/cli/v3"
)

func makeInfo(name string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: "Output applied migrations, migrations to apply",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "direction",
				Value: "forward",
				Usage: "which way to look? (forward|reverse)",
			},
			&cli.StringFlag{
				Name:  "format",
				Value: "tsv",
				Usage: "output format, one of (json|tsv)",
			},
			&cli.StringFlag{
				Name:  "version",
				Value: "",
				Usage: fmt.Sprintf("timestamp of migration, format: %s", internal.TimeFormat),
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Value: 0,
				Usage: fmt.Sprintf("max duration to run, ignored if non-positive, example vals %q", exampleDurationVals),
			},
		},
		Description: `List applied migrations, preview migrations to apply.

It's an introspection tool that can be used to show exactly which migration
versions would be applied, in either a forward or reverse direction, before
applying them.

It also takes a "direction" flag if you want to know what would be applied
in a rollback or remigrate operation. The "version" flag can be used to
limit or extend the range of migrations to apply.`,
		Action: func(ctx context.Context, c *cli.Command) error {
			driver, err := getDriver(ctx)
			if err != nil {
				return fmt.Errorf("getting driver from %s command: %w", name, err)
			}
			timeout := c.Duration("timeout")
			dirFS := os.DirFS(c.String(pathToFilesFlagname))
			migrationsTable := c.String(migrationsTableFlagname)
			direction := c.String("direction")
			version := c.String("version")
			format := c.String("format")

			return runInfo(
				ctx,
				driver,
				timeout,
				dirFS,
				migrationsTable,
				os.Stdout,
				format,
				forward(direction),
				version,
			)
		},
	}
}

func runInfo(
	ctx context.Context,
	driverConn DriverConnector,
	timeout time.Duration,
	dirFS fs.FS,
	migrationsTable string,
	w io.Writer,
	format string,
	forward bool,
	version string,
) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	err := withConnection(ctx, "", driverConn, func(ictx context.Context) error {
		return godfish.Info(ictx, driverConn, dirFS.(fs.ReadDirFS), forward, version, w, format, migrationsTable)
	})
	if errors.Is(err, godfish.ErrSchemaMigrationsMissingColumns) {
		err = fmt.Errorf("%w; run the %q command to fix this", err, upgradeCmdName)
	}
	return err
}

func forward(input string) bool {
	d := strings.ToLower(input)
	for _, prefix := range []string{"rev", "roll", "back", "down"} {
		if strings.HasPrefix(d, prefix) {
			return false
		}
	}
	return true
}
