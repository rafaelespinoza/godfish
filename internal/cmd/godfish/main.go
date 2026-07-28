// Command godfish is an omni-command of sorts. It bundles each [godfish.Driver]
// implementation into a single binary. The top-level command merely routes
// arguments to the chosen driver, which is specified as the 1st positional arg.
package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/rafaelespinoza/godfish/drivers/cassandra"
	"github.com/rafaelespinoza/godfish/drivers/mysql"
	"github.com/rafaelespinoza/godfish/drivers/postgres"
	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
	"github.com/rafaelespinoza/godfish/drivers/sqlserver"
	"github.com/rafaelespinoza/godfish/internal/cmd"

	"github.com/urfave/cli/v3"
)

func main() {
	commands := buildCommands()

	root := &cli.Command{
		Name:                  "godfish",
		Usage:                 "A delegator for all supported godfish DB drivers",
		Commands:              commands,
		EnableShellCompletion: true,
		Suggest:               true,
		Description: `This is a unified entrypoint for the DB migration manager, godfish.
Each DB driver binary is compiled within this binary,

  The upstream repository is:
    https://github.com/rafaelespinoza/godfish
  The Homebrew tap lives at:
    https://github.com/rafaelespinoza/homebrew-godfish`,
	}
	root.CommandNotFound = root.Commands[0].CommandNotFound

	if err := root.Run(context.Background(), os.Args); err != nil {
		slog.Error("running command", slog.Any("error", err))
	}
}

func buildCommands() []*cli.Command {
	driversWithSampleDSNs := []struct {
		driver    cmd.DriverConnector
		sampleDSN string
	}{
		{cassandra.NewDriver(), cassandra.SampleDSN},
		{mysql.NewDriver(), mysql.SampleDSN},
		{postgres.NewDriver(), postgres.SampleDSN},
		{sqlite3.NewDriver(), sqlite3.SampleDSN},
		{sqlserver.NewDriver(), sqlserver.SampleDSN},
	}
	driverNames := make([]string, len(driversWithSampleDSNs))
	numCommands := len(driverNames) + 1 // add version command
	commands := make([]*cli.Command, numCommands)
	for i, tuple := range driversWithSampleDSNs {
		driverNames[i] = tuple.driver.Name()
		commands[i] = newDriverCommand(tuple.driver, tuple.sampleDSN)
	}
	namer := allTheNames{name: strings.Join(driverNames, ",")}
	commands[numCommands-1] = cmd.MakeVersion("version", &namer)

	return commands
}

func newDriverCommand(dc cmd.DriverConnector, dsn string) *cli.Command {
	c := cmd.New(dc, dsn).(*cli.Command)

	c.Name = dc.Name()
	c.Suggest = true
	c.Category = "driver"
	// The parent command will enable this feature.
	c.EnableShellCompletion = false

	return c
}

// allTheNames satisfies an interface so that version metadata can mention each
// driver compiled within.
type allTheNames struct{ name string }

func (d *allTheNames) Name() string { return d.name }
