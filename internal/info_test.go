package internal_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/rafaelespinoza/godfish/internal"
)

func TestTSV(t *testing.T) {
	names := []string{"alfa", "bravo", "charlie", "delta"}

	// Set up migrations to print. The first half is considered in the past,
	// applied. The latter half is considered in the future, not yet applied.
	migrations := mustMakeMigrations(t, names...)

	t.Run("ok", func(t *testing.T) {
		var buf bytes.Buffer
		if err := printMigrations(internal.NewTSV(&buf), migrations[:2], migrations[2:]); err != nil {
			t.Fatal(err)
		}

		const numExpectedFields = 5
		expected := [][numExpectedFields]string{
			{"i", "version", "applied", "executed_at", "label"},
			{"0", "1000", "true", "1000-01-02 15:04:05", "alfa"},
			{"1", "2000", "true", "2000-01-02 15:04:05", "bravo"},
			{"2", "3000", "false", "-", "charlie"},
			{"3", "4000", "false", "-", "delta"},
		}

		tsvReader := csv.NewReader(&buf)
		tsvReader.Comma = '\t'
		tsvReader.FieldsPerRecord = numExpectedFields
		tsvReader.TrimLeadingSpace = true
		lines, err := tsvReader.ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != len(expected) {
			t.Fatalf("wrong number of lines; got %d, expected %d", len(lines), len(expected))
		}
		for i, line := range lines {
			if len(line) != numExpectedFields {
				t.Fatalf("wrong number of fields per line; got %d, expected %d", len(line), numExpectedFields)
			}
			for j := range numExpectedFields {
				got := line[j]
				exp := expected[i][j]
				if got != exp {
					t.Errorf("got %q, expected %q", got, exp)
				}
			}
		}
	})

	t.Run("error", func(t *testing.T) {
		w := errWriter{writeFn: func(p []byte) (int, error) { return len(p), errors.New("test") }}
		if err := printMigrations(internal.NewTSV(&w), migrations[:2], migrations[2:]); err != nil {
			t.Fatal("function should try to print as much as it can without erroring out")
		}
	})
}

func TestJSON(t *testing.T) {
	// migration is copied from the function under test.
	type migration struct {
		I          int    `json:"i"`
		Version    string `json:"version"`
		Applied    bool   `json:"applied"`
		ExecutedAt string `json:"executed_at"`
		Label      string `json:"label"`
	}

	names := []string{"alfa", "bravo", "charlie", "delta"}

	// Set up migrations to print. The first half is considered in the past,
	// applied. The latter half is considered in the future, not yet applied.
	migrations := mustMakeMigrations(t, names...)

	t.Run("ok", func(t *testing.T) {
		expected := []migration{
			{I: 0, Version: "1000", Applied: true, ExecutedAt: "1000-01-02 15:04:05", Label: "alfa"},
			{I: 1, Version: "2000", Applied: true, ExecutedAt: "2000-01-02 15:04:05", Label: "bravo"},
			{I: 2, Version: "3000", Applied: false, ExecutedAt: "", Label: "charlie"},
			{I: 3, Version: "4000", Applied: false, ExecutedAt: "", Label: "delta"},
		}

		var buf bytes.Buffer
		if err := printMigrations(internal.NewJSON(&buf), migrations[:2], migrations[2:]); err != nil {
			t.Fatal(err)
		}

		for i := range len(names) {
			line, ierr := buf.ReadBytes('\n')
			if ierr != nil {
				t.Fatal(ierr)
			}

			var got migration
			if err := json.Unmarshal(line, &got); err != nil {
				t.Fatal(err)
			}
			exp := expected[i]
			if got != exp {
				t.Errorf("item[%d] incorrect\ngot:      %#v\nexpected: %#v", i, got, exp)
			}
		}

		// should be no more data remaining.
		if _, err := buf.ReadBytes('\n'); err != io.EOF {
			t.Errorf("wrong error; got %v, expected %v", err, io.EOF)
		}
	})

	t.Run("error", func(t *testing.T) {
		w := errWriter{writeFn: func(p []byte) (int, error) { return len(p), errors.New("test") }}
		if err := printMigrations(internal.NewJSON(&w), migrations[:2], migrations[2:]); err != nil {
			t.Fatal("function should try to print as much as it can without erroring out")
		}
	})
}

// mustMakeMigrations creates migrations based on the input labels. The index
// position of each label in labels indirectly determines the Version,
// ExecutedAt fields of each Migration.
func mustMakeMigrations(t *testing.T, labels ...string) []*internal.Migration {
	t.Helper()

	out := make([]*internal.Migration, len(labels))
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
			mig.Applied = true
		}
		out[i] = &mig
	}
	return out
}

// printMigrations is copied from the godfish core library.
func printMigrations(p internal.InfoPrinter, applied, toApply []*internal.Migration) error {
	toPrint := make([]*internal.Migration, 0, len(applied)+len(toApply))
	toPrint = append(toPrint, applied...)
	toPrint = append(toPrint, toApply...)
	return p.PrintInfo(toPrint)
}

type errWriter struct{ writeFn func([]byte) (int, error) }

func (w *errWriter) Write(p []byte) (n int, e error) {
	if w.writeFn == nil {
		panic("define writeFn")
	}
	return w.writeFn(p)
}
