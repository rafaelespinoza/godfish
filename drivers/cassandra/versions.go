package cassandra

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/gocql/gocql"
)

// execAllAscending executes query, reads the entire results and then sorts the
// results ascendingly. The output av will be non-nil, read its err field to
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
		closeErr := scanner.Err()
		if av.err != nil && closeErr != nil {
			// These errors might be the same error, not entirely sure...
			av.err = fmt.Errorf("original err: %w; close err: %v", av.err, closeErr)
			return
		}
		av.err = closeErr
	}()

	// Read it all up front so DB resources can be closed while also avoid nil
	// access errors.
	for scanner.Next() {
		var version, label string
		if err := scanner.Scan(&version, &label); err != nil {
			av.err = err
			return &av
		}
		av.versions = append(av.versions, migration{version, label})
	}

	return &av
}

type appliedVersions struct {
	counter  int
	versions []migration
	err      error
}

func (a *appliedVersions) Close() error { return a.err }

func (a *appliedVersions) Next() bool {
	if a.err != nil {
		return false
	}
	return a.counter < len(a.versions)
}

// Scan is called by the godfish library. Unlike sql.Driver-based
// implementations, the data has already been read from the DB by the time this
// function is called. See details in the execAllAscending function.
func (a *appliedVersions) Scan(dest ...interface{}) error {
	if a.err != nil {
		return a.err
	}
	if !a.Next() {
		return nil
	}
	curr := a.versions[a.counter]

	switch val := dest[0].(type) {
	case *string:
		*val = curr.id
	default:
		return fmt.Errorf("unexpected type (%T) for %q field", val, "migration_id")
	}

	switch val := dest[1].(type) {
	case *sql.NullString:
		if err := val.Scan(curr.label); err != nil {
			return fmt.Errorf("failed to Scan %q field: %w", "label", err)
		}
	default:
		return fmt.Errorf("unexpected type (%T) for %q field", val, "label")
	}

	a.counter++
	return nil
}

type migration struct {
	id    string
	label string
}
