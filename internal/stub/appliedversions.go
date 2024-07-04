package stub

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/rafaelespinoza/godfish"
	"github.com/rafaelespinoza/godfish/internal"
)

type appliedVersions struct {
	counter  int
	versions []internal.Migration
}

// NewAppliedVersions constructs an in-memory AppliedVersions implementation for
// testing purposes.
func NewAppliedVersions(migrations ...internal.Migration) godfish.AppliedVersions {
	versions := make([]internal.Migration, len(migrations))
	copy(versions, migrations)
	return &appliedVersions{versions: versions}
}

func (r *appliedVersions) Close() error {
	r.counter = 0
	return nil
}

func (r *appliedVersions) Next() bool { return r.counter < len(r.versions) }

func (r *appliedVersions) Scan(dest ...any) (err error) {
	if len(dest) != 2 {
		err = fmt.Errorf("expected 2 args, got %d", len(dest))
		return
	}
	if !r.Next() {
		err = errors.New("no more results")
		return
	}

	curr := r.versions[r.counter]
	r.counter++

	switch val := dest[0].(type) {
	case *string:
		*val = curr.Version.String()
	default:
		return fmt.Errorf("unexpected type (%T) for %q field", val, "version")
	}

	switch val := dest[1].(type) {
	case *sql.NullString:
		if err = val.Scan(curr.Label); err != nil {
			return fmt.Errorf("failed to Scan %q field: %w", "label", err)
		}
	default:
		return fmt.Errorf("unexpected type (got %T) for %q field", val, "label")
	}

	return nil
}
