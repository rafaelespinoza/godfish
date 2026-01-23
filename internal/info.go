package internal

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"text/tabwriter"
	"time"
)

// InfoPrinter outputs the state of one migration.
type InfoPrinter interface {
	PrintInfo([]*Migration) error
}

// NewTSV constructs an InfoPrinter to write out tab separated values.
func NewTSV(w io.Writer) InfoPrinter {
	tw := tabwriter.NewWriter(w, 0, 8, 1, '\t', 0)
	return &tsvPrinter{tw}
}

// NewJSON constructs an InfoPrinter to write out JSON.
func NewJSON(w io.Writer) InfoPrinter {
	enc := json.NewEncoder(w)
	return &jsonPrinter{enc}
}

type tsvPrinter struct{ tw *tabwriter.Writer }
type jsonPrinter struct{ enc *json.Encoder }

func (p *tsvPrinter) PrintInfo(in []*Migration) error {
	const format = "%s\t%s\t%s\t%s\t%s"

	// headers
	_, err := fmt.Fprintf(p.tw, format+"\n", "i", "version", "applied", "executed_at", "label")
	if err != nil {
		slog.Error("internal: printing TSV headers", slog.Any("error", err))
	}

	// body
	var executedAt, label string
	for i, mig := range in {
		// These fields could be empty values. For display purposes in this format,
		// show a "-" rather than "" to show that data is confirmed to be empty.
		// It also eases unit testing if interpreting output with a TSV reader. When
		// the field is empty, then the tabwriter may add a dummy delimiter value
		// instead, which is fine for human visual purposes, but breaks unit tests
		// that rely on parsing TSV. For that reason, put something here.
		executedAt = cmp.Or(formatTime(mig.ExecutedAt), "-")
		label = cmp.Or(mig.Label, "-")

		_, err = fmt.Fprintf(
			p.tw,
			format+"\n",
			strconv.Itoa(i), mig.Version.String(), strconv.FormatBool(mig.Applied), executedAt, label,
		)
		if err != nil {
			slog.Error(
				"internal: printing TSV body",
				slog.Any("error", err), slog.String("version", mig.Version.String()), slog.String("label", label),
			)
		}
	}
	if err = p.tw.Flush(); err != nil {
		slog.Error("internal: flushing TSV", slog.Any("error", err))
	}
	return nil
}

func (p *jsonPrinter) PrintInfo(in []*Migration) error {
	type migration struct {
		I          int    `json:"i"`
		Version    string `json:"version"`
		Applied    bool   `json:"applied"`
		ExecutedAt string `json:"executed_at"`
		Label      string `json:"label"`
	}
	encodeJSON := p.enc.Encode
	var err error

	for i, mig := range in {
		err = encodeJSON(migration{
			I:          i,
			Version:    mig.Version.String(),
			Applied:    mig.Applied,
			ExecutedAt: formatTime(mig.ExecutedAt),
			Label:      mig.Label,
		})
		if err != nil {
			slog.Error(
				"internal: printing JSON item",
				slog.Any("error", err), slog.String("version", mig.Version.String()), slog.String("label", mig.Label),
			)
		}
	}

	return nil
}

func formatTime(t time.Time) string {
	t = t.UTC()
	if t.IsZero() {
		return ""
	}
	return t.Format(time.DateTime)
}
