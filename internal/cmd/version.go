package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/rafaelespinoza/alf"
	"github.com/rafaelespinoza/godfish/internal/version"
)

func makeVersion(name string) alf.Directive {
	var formatJSON bool

	return &alf.Command{
		Description: "show metadata about the build",
		Setup: func(p flag.FlagSet) *flag.FlagSet {
			flags := flag.NewFlagSet(name, flag.ExitOnError)
			flags.BoolVar(&formatJSON, "json", false, "format output as JSON")
			flags.Usage = func() {
				fmt.Printf(`Usage: %s [flags]

	Prints some versioning info to stdout. Pass the -json flag to get JSON.
`,
					name,
				)
				printFlagDefaults(flags)
			}
			return flags
		},
		Run: func(_ context.Context) error {
			if !formatJSON {
				fmt.Printf("BranchName:	%s\n", version.BranchName)
				fmt.Printf("BuildTime: 	%s\n", version.BuildTime)
				fmt.Printf("Driver: 	%s\n", version.Driver)
				fmt.Printf("CommitHash:	%s\n", version.CommitHash)
				fmt.Printf("GoVersion: 	%s\n", version.GoVersion)
				fmt.Printf("Tag:		%s\n", version.Tag)
				return nil
			}
			out, err := json.Marshal(
				map[string]string{
					"BranchName": version.BranchName,
					"BuildTime":  version.BuildTime,
					"Driver":     version.Driver,
					"CommitHash": version.CommitHash,
					"GoVersion":  version.GoVersion,
					"Tag":        version.Tag,
				},
			)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", out)
			return nil
		},
	}
}
