package commands

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/rafaelespinoza/godfish/internal/version"
)

var _Version = func() *subcommand {
	var formatJSON bool
	out := subcommand{
		description: "show metadata about the build",
		setup: func(a *arguments) *flag.FlagSet {
			flags := flag.NewFlagSet("version", flag.ExitOnError)
			flags.BoolVar(&formatJSON, "json", false, "format output as JSON")
			flags.Usage = func() {
				fmt.Printf(`Usage: %s version [-json]

	Prints some versioning info to stdout. Pass the -json flag to get JSON.`,
					bin,
				)
				printFlagDefaults(flags)
			}
			return flags
		},
		run: func(a arguments) error {
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
	return &out
}()
