package cassandra

import (
	"fmt"
	"sort"

	"github.com/gocql/gocql"
)

// execAllAscending executes query, reads the entire results and then sorts the
// results ascendingly. The output av will be non-nil, read its err field to
// check if an error was encountered.
func execAllAscending(query *gocql.Query) *appliedVersions {
	scanner := query.Iter().Scanner()
	av := appliedVersions{versions: make([]string, 0)}

	defer func() {
		sort.Strings(av.versions)

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
		var version string
		if err := scanner.Scan(&version); err != nil {
			av.err = err
			return &av
		}
		av.versions = append(av.versions, version)
	}

	return &av
}

type appliedVersions struct {
	counter  int
	versions []string
	err      error
}

func (a *appliedVersions) Close() error { return a.err }
func (a *appliedVersions) Next() bool {
	if a.err != nil {
		return false
	}
	return a.counter < len(a.versions)
}

func (a *appliedVersions) Scan(dest ...any) error {
	if a.err != nil {
		return a.err
	}

	out, ok := dest[0].(*string)
	if !ok {
		return fmt.Errorf("dest argument should be a %T", out)
	}
	if !a.Next() {
		return nil
	}
	*out = a.versions[a.counter]
	a.counter++
	return nil
}
