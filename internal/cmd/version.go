package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/rafaelespinoza/alf"
)

// Pieces of version metadata that can be set through -ldflags at build time.
// TODO: Look into embedding this info when building with golang >= v1.18.
var (
	versionBranchName string
	versionBuildTime  string
	versionDriver     string
	versionCommitHash string
	versionGoVersion  string
	versionTag        string
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
				fmt.Printf("BranchName:	%s\n", versionBranchName)
				fmt.Printf("BuildTime: 	%s\n", versionBuildTime)
				fmt.Printf("Driver: 	%s\n", versionDriver)
				fmt.Printf("CommitHash:	%s\n", versionCommitHash)
				fmt.Printf("GoVersion: 	%s\n", versionGoVersion)
				fmt.Printf("Tag:		%s\n", versionTag)
				return nil
			}
			out, err := json.Marshal(
				map[string]string{
					"BranchName": versionBranchName,
					"BuildTime":  versionBuildTime,
					"Driver":     versionDriver,
					"CommitHash": versionCommitHash,
					"GoVersion":  versionGoVersion,
					"Tag":        versionTag,
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
