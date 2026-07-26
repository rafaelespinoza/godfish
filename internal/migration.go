package internal

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// A Migration is a database change with a direction name and timestamp.
// Typically, a Migration with a DirForward Direction is paired with another
// migration of DirReverse that has the same label.
type Migration struct {
	Indirection Indirection
	Label       string
	Version     Version
	Applied     bool
	ExecutedAt  time.Time
	Filename    string // the file basename with an extension.
}

// ParseMigration constructs a Migration from a Filename.
func ParseMigration(name Filename) (mig *Migration, err error) {
	basename := filepath.Base(string(name))
	indirection := parseIndirection(basename)
	if indirection.Value == DirUnknown {
		err = fmt.Errorf(
			"%w; could not parse Direction for filename %q",
			ErrDataInvalid, name,
		)
		return
	}

	// index of the start of timestamp
	i := len(indirection.Label) + len(filenameDelimeter)
	version, err := ParseVersion(basename)
	if err != nil {
		err = fmt.Errorf(
			"%w, could not parse version for filename %q; %v",
			ErrDataInvalid, version, err,
		)
		return
	}

	var label string
	// index of the start of migration label
	j := i + len(version.String()) + len(filenameDelimeter)
	if j < len(basename) {
		tail := string(basename[j:])
		ext := filepath.Ext(tail)
		label = strings.TrimSuffix(tail, ext)
	}

	mig = &Migration{
		Indirection: indirection,
		Label:       label,
		Version:     version,
	}
	return
}

// ToFilename converts a Migration to a Filename.
func (m *Migration) ToFilename() Filename {
	return MakeFilename(
		m.Version.String(),
		m.Indirection,
		m.Label,
	)
}

// LogValue lets *Migration implement the [log/slog.Valuer] interface.
func (m *Migration) LogValue() slog.Value {
	ind := m.Indirection
	ver := m.Version

	var executedAt string
	if x := m.ExecutedAt; !x.IsZero() {
		executedAt = x.UTC().Format(time.RFC3339)
	}
	return slog.GroupValue(
		slog.Group("indirection", slog.String("direction", ind.Value.String()), slog.String("label", ind.Label)),
		slog.String("label", m.Label),
		slog.Group("version", slog.String("string", ver.String()), slog.Int64("value", ver.Value())),
		slog.Bool("applied", m.Applied),
		slog.String("executed_at", executedAt),
		slog.String("filename", m.Filename),
	)
}

// Migrations is a list of *Migrations.
type Migrations []*Migration

// LogValue lets the list of *Migrations implement the [log/slog.Valuer] interface.
func (m Migrations) LogValue() slog.Value {
	vals := make([]slog.Attr, len(m))
	for i, val := range m {
		vals[i] = slog.Any(strconv.Itoa(i), val.LogValue())
	}
	return slog.GroupValue(vals...)
}

// MigrationParams collects inputs needed to generate migration files. Setting
// Reversible to true will generate a migration file for each direction.
// Otherwise, it only generates a file in the forward direction. The Directory
// field refers to the path to the directory with the migration files.
type MigrationParams struct {
	Forward           Migration
	Reverse           Migration
	Reversible        bool
	Dirpath           string
	FilenameExtension string
}

// NewMigrationParams constructs a MigrationParams that's ready to use.
func NewMigrationParams(name string, reversible bool, dirpath, fwdLabel, revLabel, filenameExt string) (out *MigrationParams, err error) {
	fwdLabel = cmp.Or(fwdLabel, ForwardDirections[0])
	if err = ValidateForwardDirectionLabel(fwdLabel); err != nil {
		return
	}

	revLabel = cmp.Or(revLabel, ReverseDirections[0])
	if err = ValidateReverseDirectionLabel(revLabel); err != nil {
		return
	}

	now := time.Now().UTC()
	version := timestamp{value: now.Unix(), label: now.Format(TimeFormat)}

	out = &MigrationParams{
		Reversible: reversible,
		Dirpath:    dirpath,
		Forward: Migration{
			Indirection: Indirection{Value: DirForward, Label: fwdLabel},
			Label:       name,
			Version:     &version,
		},
		Reverse: Migration{
			Indirection: Indirection{Value: DirReverse, Label: revLabel},
			Label:       name,
			Version:     &version,
		},
		FilenameExtension: filenameExt,
	}
	return
}

// GenerateFiles creates the migration files. If the migration is reversible it
// generates files in forward and reverse directions; otherwise it generates
// just one migration file in the forward direction. It closes each file handle
// when it's done.
func (m *MigrationParams) GenerateFiles() (err error) {
	var forwardFile, reverseFile *os.File

	if forwardFile, err = newMigrationFile(m.Forward, m.Dirpath, m.FilenameExtension); err != nil {
		return
	}

	slog.Info(msgPrefix+": created forward file", slog.String("filename", forwardFile.Name()))
	defer func() { _ = forwardFile.Close() }()

	if !m.Reversible {
		slog.Info(msgPrefix + ": migration marked irreversible, did not create reverse file")
		return
	}

	if reverseFile, err = newMigrationFile(m.Reverse, m.Dirpath, m.FilenameExtension); err != nil {
		return
	}
	slog.Info(msgPrefix+": created reverse file", slog.String("filename", reverseFile.Name()))
	defer func() { _ = reverseFile.Close() }()
	return
}

func newMigrationFile(m Migration, baseDir string, ext string) (*os.File, error) {
	if len(ext) > 0 && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	name := filepath.Join(baseDir, string(m.ToFilename())+ext)
	return os.Create(filepath.Clean(name))
}
