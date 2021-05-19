package stub

import (
	"strconv"

	"github.com/rafaelespinoza/godfish"
)

type version string

// NewVersion converts the input to a Version for testing purposes.
func NewVersion(v string) godfish.Version { return version(v) }

func (v version) Before(u godfish.Version) bool {
	w := u.(version) // potential panic intended, keep tests simple
	return string(v) < string(w)
}

func (v version) String() string { return string(v) }

func (v version) Value() int64 {
	i, e := strconv.ParseInt(v.String()[:4], 10, 64)
	if e != nil {
		panic(e)
	}
	return i
}
