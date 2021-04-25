package info

import (
	"fmt"
	"io"

	"github.com/rafaelespinoza/godfish"
)

func NewTSV(w io.Writer) godfish.InfoPrinter  { return &tsvPrinter{w} }
func NewJSON(w io.Writer) godfish.InfoPrinter { return &jsonPrinter{w} }

type tsvPrinter struct{ w io.Writer }
type jsonPrinter struct{ w io.Writer }

func (p *tsvPrinter) PrintInfo(state string, mig godfish.Migration) (e error) {
	_, e = fmt.Fprintf(
		p.w,
		"%s\t%s\t%s\n",
		state, mig.Version().String(), godfish.MakeMigrationFilename(mig),
	)
	return
}

func (p *jsonPrinter) PrintInfo(state string, mig godfish.Migration) (e error) {
	_, e = fmt.Fprintf(
		p.w,
		`{"state":%q,"version":%q,"filename":%q}
`, // delimit each migration by a newline.
		state, mig.Version().String(), godfish.MakeMigrationFilename(mig),
	)
	return
}
