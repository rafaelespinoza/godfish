package stub

import "io/fs"

// FS enables easier filesystem-based tests. For best results, set the embedded
// FS to an [fstest.MapFS]. Method behavior may be specified via the function
// fields. If a function field is zero value, then the embedded FS method is
// invoked instead.
type FS struct {
	fs.FS
	OpenFn     func(string) (fs.File, error)
	ReadDirFn  func(string) ([]fs.DirEntry, error)
	ReadFileFn func(string) ([]byte, error)
}

func (f FS) Open(name string) (fs.File, error) {
	if f.OpenFn == nil {
		return f.FS.Open(name)
	}
	return f.Open(name)
}

func (f FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if f.ReadDirFn == nil {
		if dirFS, ok := f.FS.(fs.ReadDirFS); ok {
			return dirFS.ReadDir(name)
		}
		panic("define ReadDirFn")
	}
	return f.ReadDirFn(name)
}

func (f FS) ReadFile(name string) ([]byte, error) {
	if f.ReadFileFn == nil {
		if fileFS, ok := f.FS.(fs.ReadFileFS); ok {
			return fileFS.ReadFile(name)
		}
		panic("define ReadFileFn")
	}
	return f.ReadFileFn(name)
}

// Ensure that both a value or pointer to value may be used in library tests.
var (
	_ fs.ReadDirFS = (*FS)(nil)
	_ fs.ReadDirFS = FS{}

	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.ReadFileFS = FS{}
)
