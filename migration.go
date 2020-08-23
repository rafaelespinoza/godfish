package godfish

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A Migration is a database change with a direction name and timestamp.
// Typically, a Migration with a DirForward Direction is paired with another
// migration of DirReverse that has the same label.
type Migration interface {
	Indirection() Indirection
	Label() string
	Version() Version
}

// mutation implements the Migration interface.
type mutation struct {
	indirection Indirection
	label       string
	version     Version
}

var _ Migration = (*mutation)(nil)

func (m *mutation) Indirection() Indirection { return m.indirection }
func (m *mutation) Label() string            { return m.label }
func (m *mutation) Version() Version         { return m.version }

func parseMigration(name filename) (mig Migration, err error) {
	basename := filepath.Base(string(name))
	indirection := parseIndirection(basename)
	if indirection.Value == DirUnknown {
		err = fmt.Errorf(
			"%w; could not parse Direction for filename %q",
			errDataInvalid, name,
		)
		return
	}

	// index of the start of timestamp
	i := len(indirection.Label) + len(filenameDelimeter)
	version, err := parseVersion(basename)
	if err != nil {
		err = fmt.Errorf(
			"%w, could not parse version for filename %q; %v",
			errDataInvalid, version, err,
		)
		return
	}

	// index of the start of migration label
	j := i + len(version.String()) + len(filenameDelimeter)
	mig = &mutation{
		indirection: indirection,
		label:       strings.TrimSuffix(string(basename[j:]), ".sql"),
		version:     version,
	}
	return
}

// MigrationParams collects inputs needed to generate migration files. Setting
// Reversible to true will generate a migration file for each direction.
// Otherwise, it only generates a file in the forward direction. The Directory
// field refers to the path to the directory with the migration files.
type MigrationParams struct {
	Forward    Migration
	Reverse    Migration
	Reversible bool
	Directory  *os.File
}

// NewMigrationParams constructs a MigrationParams that's ready to use. Passing
// in true for reversible means that a complementary SQL file will be made for
// rolling back. The directory parameter specifies which directory to output the
// files to.
func NewMigrationParams(label string, reversible bool, directory *os.File) (*MigrationParams, error) {
	var out MigrationParams
	var err error
	var info os.FileInfo
	if info, err = directory.Stat(); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("input dir %q should be a directory", info.Name())
	}
	out.Directory = directory

	out.Reversible = reversible
	now := time.Now().UTC()
	version := &timestamp{value: now.Unix(), label: now.Format(TimeFormat)}
	out.Forward = &mutation{
		indirection: Indirection{Value: DirForward, Label: forwardDirections[0]},
		label:       label,
		version:     version,
	}
	out.Reverse = &mutation{
		indirection: Indirection{Value: DirReverse, Label: reverseDirections[0]},
		label:       label,
		version:     version,
	}
	return &out, nil
}

// GenerateFiles creates the migration files. If the migration is reversible it
// generates files in forward and reverse directions; otherwise is generates
// just one migration file in the forward direction. It closes each file handle
// when it's done.
func (m *MigrationParams) GenerateFiles() (err error) {
	var forwardFile, reverseFile *os.File
	defer func() {
		forwardFile.Close()
		reverseFile.Close()
	}()
	baseDir := m.Directory.Name()
	if forwardFile, err = newMigrationFile(m.Forward, baseDir); err != nil {
		return
	}
	fmt.Println("created forward file:", forwardFile.Name())
	if !m.Reversible {
		fmt.Println("migration marked irreversible, did not create reverse file")
		return
	}
	if reverseFile, err = newMigrationFile(m.Reverse, baseDir); err != nil {
		return
	}
	fmt.Println("created reverse file:", reverseFile.Name())
	return
}

func newMigrationFile(m Migration, baseDir string) (*os.File, error) {
	return os.Create(baseDir + "/" + makeMigrationFilename(m))
}

// makeMigrationFilename passes in a Migration's fields to create a filename.
func makeMigrationFilename(m Migration) string {
	return makeFilename(
		m.Version().String(),
		m.Indirection(),
		m.Label(),
	)
}
