package internal

import "strings"

const filenameDelimeter = "-"

// Filename is just a string with a specific format to migration files. One part
// has a generated timestamp, one part has a direction, another has a label.
type Filename string

// MakeFilename creates a filename based on the independent parts.
//
// Format:
//
//	"${direction}-${version}-${label}"
func MakeFilename(version string, indirection Indirection, label string) Filename {
	dir := strings.ToLower(indirection.Label) + filenameDelimeter

	// the length will top out at the high quantifier for this regexp.
	ver := timeformatMatcher.FindString(version) + filenameDelimeter
	return Filename(dir + ver + label)
}
