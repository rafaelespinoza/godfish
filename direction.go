package godfish

import "strings"

// Direction describes which way the change goes.
type Direction uint8

const (
	// DirUnknown is a fallback value for an undetermined direction.
	DirUnknown Direction = iota
	// DirForward is like migrate up, typically the change you want to apply to
	// the DB.
	DirForward
	// DirReverse is like migrate down; used for rollbacks. Not all changes can
	// be rolled back.
	DirReverse
)

func (d Direction) String() string {
	return [...]string{"Unknown", "Forward", "Reverse"}[d]
}

var (
	forwardDirections = []string{
		strings.ToLower(DirForward.String()),
		"migrate",
		"up",
	}
	reverseDirections = []string{
		strings.ToLower(DirReverse.String()),
		"rollback",
		"down",
	}
)

// Indirection is some glue to help determine the direction of a migration,
// usually from a filename with an alias for a direction.
type Indirection struct {
	Value Direction
	Label string
}

func parseIndirection(basename string) (ind Indirection) {
	lo := strings.ToLower(basename)
	for _, pre := range forwardDirections {
		if strings.HasPrefix(lo, pre) {
			ind.Value = DirForward
			ind.Label = pre
			return
		}
	}
	for _, pre := range reverseDirections {
		if strings.HasPrefix(lo, pre) {
			ind.Value = DirReverse
			ind.Label = pre
			return
		}
	}
	return
}
