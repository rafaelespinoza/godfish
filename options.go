package godfish

import (
	"errors"
	"fmt"
	"io"

	"github.com/rafaelespinoza/godfish/internal"
)

// options are configuration parameters set through an [opter].
type options struct {
	format          string
	migrationsTable string
	targetVersion   string
	writer          io.Writer

	// relevant for create migration

	filenameExt  string
	forwardLabel string
	reverseLabel string
}

// An Opter configures a migration/rollback and returns a validation error,
// if applicable.
type Opter interface{ setOption(opt *options) error }

type opter struct{ set func(opt *options) error }

func (o *opter) setOption(opt *options) error { return o.set(opt) }

var errNonZeroValueRequired = errors.New("non-zero value is required")

// WithMigrationsTable sets the DB table name for storing migration state.
// A zero value t is invalid and will lead to an error.
func WithMigrationsTable(t string) Opter {
	return &opter{set: func(opt *options) error {
		if t == "" {
			return fmt.Errorf("%s: %w", "WithMigrationsTable", errNonZeroValueRequired)
		}
		opt.migrationsTable = t
		return nil
	}}
}

// WithTargetVersion sets the migration version to target.
// A zero value v is invalid and will lead to an error.
func WithTargetVersion(v string) Opter {
	return &opter{set: func(opt *options) error {
		if v == "" {
			return fmt.Errorf("%s: %w", "WithTargetVersion", errNonZeroValueRequired)
		}
		opt.targetVersion = v
		return nil
	}}
}

// WithFormat sets an output format.
// A zero value f is invalid and will lead to an error.
func WithFormat(f string) Opter {
	return &opter{set: func(opt *options) error {
		if f == "" {
			return fmt.Errorf("%s: %w", "WithFormat", errNonZeroValueRequired)
		}
		opt.format = f
		return nil
	}}
}

// WithWriter sets the output writer.
// A zero value w is invalid and will lead to an error.
func WithWriter(w io.Writer) Opter {
	return &opter{set: func(opt *options) error {
		if w == nil {
			return fmt.Errorf("%s: %w", "WithWriter", errNonZeroValueRequired)
		}
		opt.writer = w
		return nil
	}}
}

// WithForwardLabel sets the name of the forward direction for when creating
// a migration file.
// Valid values for l are "forward", "migration", "up".
func WithForwardLabel(l string) Opter {
	return &opter{set: func(opt *options) error {
		if err := internal.ValidateForwardDirectionLabel(l); err != nil {
			return fmt.Errorf("%s: %w", "WithForwardLabel", err)
		}
		opt.forwardLabel = l
		return nil
	}}
}

// WithReverseLabel sets the name of the reverse direction for when creating
// a migration file.
// Valid values for l are "reverse", "rollback", "down".
func WithReverseLabel(l string) Opter {
	return &opter{set: func(opt *options) error {
		if err := internal.ValidateReverseDirectionLabel(l); err != nil {
			return fmt.Errorf("%s: %w", "WithReverseLabel", err)
		}
		opt.reverseLabel = l
		return nil
	}}
}

// WithFilenameExtension sets the name of the extension of the filename when
// creating a migration file.
func WithFilenameExtension(e string) Opter {
	return &opter{set: func(opt *options) error {
		// For now, no validations. Allow zero value strings as well.
		opt.filenameExt = e
		return nil
	}}
}

func setOptions(opts ...Opter) (*options, error) {
	options := &options{}
	var err error
	for _, opt := range opts {
		if err = opt.setOption(options); err != nil {
			return nil, err
		}
	}
	return options, nil
}
