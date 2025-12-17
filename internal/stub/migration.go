package stub

import "github.com/rafaelespinoza/godfish/internal"

// NewMigration constructs a migration that can be used to override the version
// field so that the generated filename is unique enough for testing purposes.
func NewMigration(mig internal.Migration, version internal.Version, ind internal.Indirection) *internal.Migration {
	stub := internal.Migration{
		Indirection: mig.Indirection,
		Label:       mig.Label,
		Version:     version,
	}
	if ind.Label != "" {
		stub.Indirection.Label = ind.Label
	}
	return &stub
}
