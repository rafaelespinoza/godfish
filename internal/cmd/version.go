package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/rafaelespinoza/godfish"

	"github.com/urfave/cli/v3"
)

// Pieces of version metadata that can be set through -ldflags at build time.
// TODO: Look into embedding this info when building with golang >= v1.18.
var (
	versionBranchName string
	versionBuildTime  string
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
			driver, err := getDriver(ctx)
			if err != nil {
				return fmt.Errorf("getting driver from %s command: %w", name, err)
			}
			return runVersion(driver, c.Bool("json"), os.Stdout)
		},
	}
}

func runVersion(driver godfish.Driver, outputJSON bool, w io.Writer) error {
	versionData := []struct{ Key, Val string }{
		{"BranchName", versionBranchName},
		{"BuildTime", versionBuildTime},
		{"Driver", driver.Name()},
		{"CommitHash", versionCommitHash},
		{"GoVersion", versionGoVersion},
		{"Tag", versionTag},
	}

	if !outputJSON {
		tw := tabwriter.NewWriter(w, 8, 4, 1, '\t', 0)
		for _, tuple := range versionData {
			_, _ = fmt.Fprintf(tw, "%s:\t%s\n", tuple.Key, tuple.Val)
		}
		return tw.Flush()
	}

	versionDataMap := make(map[string]string, len(versionData))
	for _, tuple := range versionData {
		versionDataMap[tuple.Key] = tuple.Val
	}

	out, err := json.Marshal(versionDataMap)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "%s\n", out)
	return nil
}
