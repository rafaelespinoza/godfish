package internal_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rafaelespinoza/godfish/internal"
)

func TestTSV(t *testing.T) {
	var buf bytes.Buffer
	names := []string{"alfa", "bravo", "charlie", "delta"}

	// Set up migrations to print. The first half is considered in the past,
	// applied. The latter half is considered in the future, not yet applied.
	migrations := mustMakeMigrations(t, names...)
	if err := printMigrations(internal.NewTSV(&buf), "up", migrations[:2]); err != nil {
		t.Fatal(err)
	}
	if err := printMigrations(internal.NewTSV(&buf), "down", migrations[2:]); err != nil {
		t.Fatal(err)
	}

	const numExpectedParts = 4
	expected := [][numExpectedParts]string{
		{"up", "1000", "1000-01-02 15:04:05", "alfa"},
		{"up", "2000", "2000-01-02 15:04:05", "bravo"},
		{"down", "3000", "", "charlie"},
		{"down", "4000", "", "delta"},
	}

	for i := range len(names) {
		line, ierr := buf.ReadString('\n')
		if ierr != nil {
			t.Fatal(ierr)
		}

		parts := strings.Split(line, "\t")
		if len(parts) != numExpectedParts {
			t.Fatalf("wrong number of parts per line; got %d, expected %d", len(parts), numExpectedParts)
		}
		for j := range numExpectedParts {
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

	// Set up migrations to print. The first half is considered in the past,
	// applied. The latter half is considered in the future, not yet applied.
	migrations := mustMakeMigrations(t, names...)
	if err := printMigrations(internal.NewJSON(&buf), "up", migrations[:2]); err != nil {
		t.Fatal(err)
	}
	if err := printMigrations(internal.NewJSON(&buf), "down", migrations[2:]); err != nil {
		t.Fatal(err)
	}

	expected := []map[string]string{
		{"state": "up", "version": "1000", "executed_at": "1000-01-02 15:04:05", "label": "alfa"},
		{"state": "up", "version": "2000", "executed_at": "2000-01-02 15:04:05", "label": "bravo"},
		{"state": "down", "version": "3000", "executed_at": "", "label": "charlie"},
		{"state": "down", "version": "4000", "executed_at": "", "label": "delta"},
	}

	for i := range len(names) {
		line, ierr := buf.ReadBytes('\n')
		if ierr != nil {
			t.Fatal(ierr)
		}

		var data map[string]string
		if err := json.Unmarshal(line, &data); err != nil {
			t.Fatal(err)
		}
		expectedKeys := []string{"state", "version", "executed_at", "label"}
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

// mustMakeMigrations creates migrations based on the input labels. The index
// position of each label in labels indirectly determines the Version,
// ExecutedAt fields of each Migration.
func mustMakeMigrations(t *testing.T, labels ...string) []internal.Migration {
	t.Helper()

	out := make([]internal.Migration, len(labels))
	now := time.Now()

	for i, label := range labels {
		num := (i + 1) * 1000
		version, err := internal.ParseVersion(strconv.Itoa(num))
		if err != nil {
			t.Fatalf("error on names[%d]: %v", i, err)
		}

		mig := internal.Migration{
			Indirection: internal.Indirection{Label: "forward", Value: internal.DirForward},
			Label:       label,
			Version:     version,
		}
		executedAt := time.Date(num, time.January, 2, 15, 4, 5, 0, time.UTC)
		if executedAt.Before(now) {
			mig.ExecutedAt = executedAt
		}
		out[i] = mig
	}
	return out
}

func printMigrations(p internal.InfoPrinter, state string, migrations []internal.Migration) (err error) {
	for i, mig := range migrations {
		if err = p.PrintInfo(state, mig); err != nil {
			err = fmt.Errorf("%w; item %d", err, i)
			return
		}
	}
	return
}
