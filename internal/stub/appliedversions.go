package stub

import (
	"fmt"

	"github.com/rafaelespinoza/godfish"
)

type appliedVersions struct {
	counter  int
	versions []string
}

// NewAppliedVersions constructs an in-memory AppliedVersions implementation for
// testing purposes.
func NewAppliedVersions(migrations ...godfish.Migration) godfish.AppliedVersions {
	out := appliedVersions{
		versions: make([]string, len(migrations)),
	}
	for i, mig := range migrations {
		out.versions[i] = mig.Version().String()
	}
	return &out
}

func (r *appliedVersions) Close() error {
	r.counter = 0
	return nil
}

func (r *appliedVersions) Next() bool { return r.counter < len(r.versions) }

func (r *appliedVersions) Scan(dest ...interface{}) error {
	var out *string
	if s, ok := dest[0].(*string); !ok {
		return fmt.Errorf("pass in *string; got %T", s)
	} else if !r.Next() {
		return fmt.Errorf("no more results")
	} else {
		out = s
	}
	*out = r.versions[r.counter]
	r.counter++
	return nil
}
