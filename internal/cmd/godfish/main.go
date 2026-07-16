// Command godfish is an omni-command of sorts. It bundles each [godfish.Driver]
// implementation into a single binary. The top-level command merely routes
// arguments to the chosen driver, which is specified as the 1st positional arg.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/rafaelespinoza/godfish/drivers/cassandra"
	"github.com/rafaelespinoza/godfish/drivers/mysql"
	"github.com/rafaelespinoza/godfish/drivers/postgres"
	"github.com/rafaelespinoza/godfish/drivers/sqlite3"
	"github.com/rafaelespinoza/godfish/drivers/sqlserver"
	"github.com/rafaelespinoza/godfish/internal/cmd"

	"github.com/urfave/cli/v3"
)

func main() {
	root := &cli.Command{
		Name:  "godfish",
		Usage: "A delegator for all supported godfish DB drivers",
		Commands: []*cli.Command{
			newDriverCommand(cassandra.NewDriver(), cassandra.SampleDSN),
			newDriverCommand(mysql.NewDriver(), mysql.SampleDSN),
			newDriverCommand(postgres.NewDriver(), postgres.SampleDSN),
			newDriverCommand(sqlite3.NewDriver(), sqlite3.SampleDSN),
			newDriverCommand(sqlserver.NewDriver(), sqlserver.SampleDSN),
		},
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

func newDriverCommand(dc cmd.DriverConnector, dsn string) *cli.Command {
	c := cmd.New(dc, dsn).(*cli.Command)

	c.Name = dc.Name()

	return c
}
