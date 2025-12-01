package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/rafaelespinoza/alf"
	"github.com/rafaelespinoza/godfish"
)

func makeInit(name string) alf.Directive {
	var conf string

	return &alf.Command{
		Description: "create godfish configuration file",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags := newFlagSet(name)
			flags.StringVar(
				&conf,
				"conf",
				".godfish.json",
				"path to godfish config file",
			)
			flags.Usage = func() {
				_, _ = fmt.Fprintf(flags.Output(), `Usage: %s [godfish-flags] %s [%s-flags]

	Creates a configuration file, unless it already exists.
`,
					bin, name, name)
				printFlagDefaults(&p)
				printFlagDefaults(flags)
			}

			return flags
		},
		Run: func(_ context.Context) error {
			return godfish.Init(conf)
		},
	}
}
