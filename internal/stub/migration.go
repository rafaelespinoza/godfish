package stub

import "github.com/rafaelespinoza/godfish/internal"

type migration struct {
	indirection internal.Indirection
	label       string
	version     internal.Version
}

// NewMigration constructs a migration that can be used to override the version
// field so that the generated filename is unique enough for testing purposes.
func NewMigration(mig internal.Migration, version internal.Version, ind internal.Indirection) internal.Migration {
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

func (m *migration) Indirection() internal.Indirection { return m.indirection }
func (m *migration) Label() string                     { return m.label }
func (m *migration) Version() internal.Version         { return m.version }
