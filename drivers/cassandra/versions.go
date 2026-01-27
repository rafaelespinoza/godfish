package cassandra

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/gocql/gocql"
)

// execAllAscending executes query, reads the entire results and then sorts the
// results ascendingly. The output av will be non-nil, read its error fields to
// check if an error was encountered.
func execAllAscending(query *gocql.Query) *appliedVersions {
	scanner := query.Iter().Scanner()
	av := appliedVersions{versions: make([]migration, 0)}

	defer func() {
		sort.Slice(av.versions, func(i, j int) bool {
			return av.versions[i].id < av.versions[j].id
		})

		// The Err method also releases resources. The scanner should not be
		// used after this point.
		av.closingErr = scanner.Err()
	}()

	// Read it all up front so DB resources can be closed while also avoid nil
	// access errors.
	for scanner.Next() {
		var version, label string
		var executedAt int64
		if err := scanner.Scan(&version, &label, &executedAt); err != nil {
			av.scanningErr = err
			return &av
		}
		av.versions = append(av.versions, migration{version, label, executedAt})
		slog.Debug(
			msgPrefix+"scanned version",
			slog.String("version", version), slog.String("label", label), slog.Int64("executed_at", executedAt),
		)
	}

	return &av
}

type appliedVersions struct {
	counter  int
	versions []migration
	// closingErr may hold an error from closing the scanner via the `Scanner.Err`
	// method, which also releases resources. An error here is more likely to
	// indicate an infrastructual problem than the scanningErr field.
	closingErr error
	// scanningErr may hold an error from scanning a result row via the method
	// `Scanner.Scan`. An error here is more likely indicates an issue with
	// the application code than the closingErr field.
	scanningErr error
}

func (a *appliedVersions) Close() error { return a.closingErr }

func (a *appliedVersions) Next() bool {
	if a.scanningErr != nil {
		return false
	}
	return a.counter < len(a.versions)
}

// Scan is called by the godfish library. Unlike sql.Driver-based
// implementations, the data has already been read from the DB by the time this
// function is called. See details in the execAllAscending function.
func (a *appliedVersions) Scan(dest ...any) error {
	if a.scanningErr != nil {
		return a.scanningErr
	}
	if !a.Next() {
		return nil
	}
	curr := a.versions[a.counter]
	a.counter++

	switch val := dest[0].(type) {
	case *string:
		*val = curr.id
	default:
		return fmt.Errorf("unexpected type (%T) for %q field", val, "migration_id")
	}

	switch val := dest[1].(type) {
	case *string:
		*val = curr.label
	default:
		return fmt.Errorf("unexpected type (%T) for %q field", val, "label")
	}

	switch val := dest[2].(type) {
	case *int64:
		*val = curr.executedAt
	default:
		return fmt.Errorf("unexpected type (%T) for %q field", val, "executed_at")
	}

	return nil
}

type migration struct {
	id         string
	label      string
	executedAt int64
}
