package cmd

import (
	"context"

	"github.com/rafaelespinoza/godfish"

	"github.com/urfave/cli/v3"
)

func makeInit(name string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: "Create godfish configuration file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "conf",
				Value:     ".godfish.json",
				Usage:     "path to godfish config file",
				TakesFile: true,
				// Local is set to true to tell the cli library to not use the global flag
				// of the same name.
				Local: true,
			},
		},
		Description: `Creates a configuration file, unless it already exists.`,
		Action: func(_ context.Context, c *cli.Command) error {
			return godfish.Init(c.String("conf"))
		},
	}
}
