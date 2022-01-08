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
	Dirpath    string
	directory  *os.File
}

// NewMigrationParams constructs a MigrationParams that's ready to use. Passing
// in true for reversible means that a complementary SQL file will be made for
// rolling back. The dirpath is the path to the directory for the files. An
// error is returned if dirpath doesn't actually represent a directory. Names
// for directions in the filename could be overridden from their default values
// (forward and reverse) with the input vars fwdLabel, revLabel when non-empty.
func NewMigrationParams(name string, reversible bool, dirpath, fwdLabel, revLabel string) (*MigrationParams, error) {
	var (
		err       error
		info      os.FileInfo
		directory *os.File
		now       time.Time
		version   timestamp
	)

	if directory, err = os.Open(dirpath); err != nil {
		return nil, err
	}
	defer directory.Close()
	if info, err = directory.Stat(); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("dirpath %q should be a path to a directory", dirpath)
	}

	now = time.Now().UTC()
	version = timestamp{value: now.Unix(), label: now.Format(TimeFormat)}

	if fwdLabel == "" {
		fwdLabel = ForwardDirections[0]
	}
	if revLabel == "" {
		revLabel = ReverseDirections[0]
	}
	return &MigrationParams{
		Reversible: reversible,
		Dirpath:    dirpath,
		Forward: &mutation{
			indirection: Indirection{Value: DirForward, Label: fwdLabel},
			label:       name,
			version:     &version,
		},
		Reverse: &mutation{
			indirection: Indirection{Value: DirReverse, Label: revLabel},
			label:       name,
			version:     &version,
		},
		directory: directory,
	}, nil
}

// GenerateFiles creates the migration files. If the migration is reversible it
// generates files in forward and reverse directions; otherwise it generates
// just one migration file in the forward direction. It closes each file handle
// when it's done.
func (m *MigrationParams) GenerateFiles() (err error) {
	var forwardFile, reverseFile *os.File
	defer func() {
		forwardFile.Close()
		reverseFile.Close()
	}()
	baseDir := m.directory.Name()

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
	return os.Create(baseDir + "/" + MakeMigrationFilename(m))
}

// MakeMigrationFilename converts a Migration m to a filename.
func MakeMigrationFilename(m Migration) string {
	return makeFilename(
		m.Version().String(),
		m.Indirection(),
		m.Label(),
	)
}
