package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v3"
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

func makeVersion(name string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: "Show metadata about the build",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Value: false,
				Usage: "format output as JSON",
			},
		},
		Description: `Prints some versioning info to stdout. Pass the -json flag to get JSON.`,
		Action: func(ctx context.Context, c *cli.Command) error {
			return runVersion(ctx, c.Bool("json"), os.Stdout)
		},
	}
}

func runVersion(_ context.Context, outputJSON bool, w io.Writer) error {
	if !outputJSON {
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
}
