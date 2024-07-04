package stub

import "github.com/rafaelespinoza/godfish/internal"

type migration struct {
	internal.Migration
	version internal.Version
}

// NewMigration constructs a migration that can be used to override the version
// field so that the generated filename is unique enough for testing purposes.
func NewMigration(mig internal.Migration, version internal.Version, _ internal.Indirection) internal.Migration { // TODO: remove Indirection from func signature
	stub := migration{
		Migration: mig,
		version:   version,
	}
	return &stub
}

func (m *migration) Version() internal.Version { return m.version }
