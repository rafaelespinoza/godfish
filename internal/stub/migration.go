package stub

import "github.com/rafaelespinoza/godfish"

type migration struct {
	indirection godfish.Indirection
	label       string
	version     godfish.Version
}

// NewMigration constructs a migration that can be used to override the version
// field so that the generated filename is unique enough for testing purposes.
func NewMigration(mig godfish.Migration, version godfish.Version, ind godfish.Indirection) godfish.Migration {
	stub := migration{
		indirection: mig.Indirection(),
		label:       mig.Label(),
		version:     version,
	}
	if ind.Label != "" {
		stub.indirection.Label = ind.Label
	}
	return &stub
}

func (m *migration) Indirection() godfish.Indirection { return m.indirection }
func (m *migration) Label() string                    { return m.label }
func (m *migration) Version() godfish.Version         { return m.version }
