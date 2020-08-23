package godfish

import (
	"regexp"
	"strconv"
	"time"
)

// Version is for comparing migrations to each other.
type Version interface {
	Before(u Version) bool
	String() string
	Value() int64
}

type timestamp struct {
	value int64
	label string
}

var _ Version = (*timestamp)(nil)

func (v *timestamp) Before(u Version) bool {
	// Until there's more than 1 interface implementation, this is fine. So,
	// panic here?  Yeah, maybe. Fail loudly, not silently.
	w := u.(*timestamp)
	return v.value < w.value
}

func (v *timestamp) String() string {
	if v.label == "" {
		return strconv.FormatInt(int64(v.value), 10)
	}
	return v.label
}

func (v *timestamp) Value() int64 { return v.value }

const (
	// TimeFormat provides a consistent timestamp layout for migrations.
	TimeFormat = "20060102150405"

	unixTimestampSecLen = len("1574079194")
)

var (
	minVersion = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Format(TimeFormat)
	maxVersion = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC).Format(TimeFormat)
)

var timeformatMatcher = regexp.MustCompile(`\d{4,14}`)

func parseVersion(basename string) (version Version, err error) {
	written := timeformatMatcher.FindString(basename)
	if ts, perr := time.Parse(TimeFormat, written); perr != nil {
		err = perr // keep going
	} else {
		version = &timestamp{value: ts.UTC().Unix(), label: written}
		return
	}

	if perr, ok := err.(*time.ParseError); ok {
		if len(perr.Value) < len(TimeFormat) {
			ts, qerr := time.Parse(TimeFormat[:len(perr.Value)], perr.Value)
			if qerr == nil {
				version = &timestamp{value: ts.UTC().Unix(), label: perr.Value}
				err = nil
				return
			}
		}
	}

	// try parsing as unix epoch timestamp
	num, err := strconv.ParseInt(written[:unixTimestampSecLen], 10, 64)
	if err != nil {
		return
	}
	version = &timestamp{value: num, label: written}
	return
}
