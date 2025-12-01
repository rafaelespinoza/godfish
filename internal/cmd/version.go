package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

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
				_, _ = fmt.Fprintf(flags.Output(), `Usage: %s [flags]

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
				tw := tabwriter.NewWriter(w, 8, 4, 1, '\t', 0)
				tuples := []struct{ key, val string }{
					{"BranchName", versionBranchName},
					{"BuildTime", versionBuildTime},
					{"Driver", versionDriver},
					{"CommitHash", versionCommitHash},
					{"GoVersion", versionGoVersion},
					{"Tag", versionTag},
				}
				for _, tuple := range tuples {
					_, _ = fmt.Fprintf(tw, "%s:\t%s\n", tuple.key, tuple.val)
				}
				return tw.Flush()
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
			_, _ = fmt.Fprintf(w, "%s\n", out)
			return nil
		},
	}
}
