package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

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
			flags := newFlagSet(name)
			flags.BoolVar(&formatJSON, "json", false, "format output as JSON")
			flags.Usage = func() {
				fmt.Fprintf(flags.Output(), `Usage: %s [flags]

	Prints some versioning info to stdout. Pass the -json flag to get JSON.
`,
					name,
				)
				printFlagDefaults(flags)
			}
			return flags
		},
		Run: func(_ context.Context) error {
			// Calling fmt.Print* also writes to stdout, but want to be explicit
			// about where this subcommand output goes.
			w := os.Stdout
			if !formatJSON {
				fmt.Fprintf(w, "BranchName:	%s\n", versionBranchName)
				fmt.Fprintf(w, "BuildTime: 	%s\n", versionBuildTime)
				fmt.Fprintf(w, "Driver: 	%s\n", versionDriver)
				fmt.Fprintf(w, "CommitHash:	%s\n", versionCommitHash)
				fmt.Fprintf(w, "GoVersion: 	%s\n", versionGoVersion)
				fmt.Fprintf(w, "Tag:		%s\n", versionTag)
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
			fmt.Fprintf(w, "%s\n", out)
			return nil
		},
	}
}
