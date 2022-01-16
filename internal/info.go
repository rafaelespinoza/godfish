package internal

import (
	"fmt"
	"io"
)

// InfoPrinter outputs the state of one migration.
type InfoPrinter interface {
	PrintInfo(state string, migration Migration) error
}

// NewTSV constructs an InfoPrinter to write out tab separated values.
func NewTSV(w io.Writer) InfoPrinter { return &tsvPrinter{w} }

// NewJSON constructs an InfoPrinter to write out JSON.
func NewJSON(w io.Writer) InfoPrinter { return &jsonPrinter{w} }

type tsvPrinter struct{ w io.Writer }
type jsonPrinter struct{ w io.Writer }

func (p *tsvPrinter) PrintInfo(state string, mig Migration) (e error) {
	_, e = fmt.Fprintf(
		p.w,
		"%s\t%s\t%s\n",
		state, mig.Version().String(), MakeMigrationFilename(mig),
	)
	return
}

func (p *jsonPrinter) PrintInfo(state string, mig Migration) (e error) {
	_, e = fmt.Fprintf(
		p.w,
		`{"state":%q,"version":%q,"filename":%q}
`, // delimit each migration by a newline.
		state, mig.Version().String(), MakeMigrationFilename(mig),
	)
	return
}
