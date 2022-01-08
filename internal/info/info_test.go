package info_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal/info"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

func TestTSV(t *testing.T) {
	var buf bytes.Buffer
	names := []string{"alfa", "bravo", "charlie", "delta"}

	if err := printMigrations(info.NewTSV(&buf), "up", mustMakeMigrations(names...)); err != nil {
		t.Fatal(err)
	}

	const numExpectedParts = 3
	expected := [][numExpectedParts]string{
		{"up", "1000", "forward-1000-alfa.sql"},
		{"up", "2000", "forward-2000-bravo.sql"},
		{"up", "3000", "forward-3000-charlie.sql"},
		{"up", "4000", "forward-4000-delta.sql"},
	}

	for i := 0; i < len(names); i++ {
		line, ierr := buf.ReadString('\n')
		switch ierr {
		case nil:
			break
		default:
			t.Fatal(ierr)
		}

		parts := strings.Split(line, "\t")
		if len(parts) != numExpectedParts {
			t.Fatalf("wrong number of parts per line; got %d, expected %d", len(parts), numExpectedParts)
		}
		for j := 0; j < numExpectedParts; j++ {
			// this newline only appears in the last item, but it's annoying.
			got := strings.TrimSuffix(parts[j], "\n")
			if got != expected[i][j] {
				t.Errorf("got %q, expected %q", got, expected[i][j])
			}
		}
	}

	// should be no more data remaining.
	if _, err := buf.ReadBytes('\n'); err != io.EOF {
		t.Errorf("wrong error; got %v, expected %v", err, io.EOF)
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	names := []string{"alfa", "bravo", "charlie", "delta"}

	if err := printMigrations(info.NewJSON(&buf), "up", mustMakeMigrations(names...)); err != nil {
		t.Fatal(err)
	}

	expected := []map[string]string{
		{"state": "up", "version": "1000", "filename": "forward-1000-alfa.sql"},
		{"state": "up", "version": "2000", "filename": "forward-2000-bravo.sql"},
		{"state": "up", "version": "3000", "filename": "forward-3000-charlie.sql"},
		{"state": "up", "version": "4000", "filename": "forward-4000-delta.sql"},
	}

	for i := 0; i < len(names); i++ {
		line, ierr := buf.ReadBytes('\n')
		switch ierr {
		case nil:
			break
		default:
			t.Fatal(ierr)
		}

		var data map[string]string
		if err := json.Unmarshal(line, &data); err != nil {
			t.Fatal(err)
		}
		expectedKeys := []string{"state", "version", "filename"}
		if len(data) != len(expectedKeys) {
			t.Errorf("wrong number of keys; got %d, expected %d", len(data), len(expectedKeys))
		}

		for _, key := range expectedKeys {
			got := data[key]
			if got != expected[i][key] {
				t.Errorf("wrong %q; got %q, expected %q", key, got, expected[i][key])
			}
		}
	}

	// should be no more data remaining.
	if _, err := buf.ReadBytes('\n'); err != io.EOF {
		t.Errorf("wrong error; got %v, expected %v", err, io.EOF)
	}
}

func mustMakeMigrations(names ...string) []godfish.Migration {
	dir, err := os.MkdirTemp(os.TempDir(), "godfish_test_*")
	if err != nil {
		panic(err)
	}

	out := make([]godfish.Migration, len(names))

	for i := 0; i < len(names); i++ {
		params, err := godfish.NewMigrationParams(names[i], false, dir, "", "")
		if err != nil {
			panic(err)
		}
		version := stub.NewVersion(strconv.Itoa((i + 1) * 1000))
		out[i] = stub.NewMigration(params.Forward, version, godfish.Indirection{})
	}
	return out
}

func printMigrations(p godfish.InfoPrinter, state string, migrations []godfish.Migration) (err error) {
	for i, mig := range migrations {
		if err = p.PrintInfo(state, mig); err != nil {
			err = fmt.Errorf("%w; item %d", err, i)
			return
		}
	}
	return
}
