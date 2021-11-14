package pep427

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/datawire/ocibuild/pkg/fsutil"
)

type skipReader struct {
	skip  int
	inner io.Reader
}

func (r *skipReader) Read(p []byte) (int, error) {
	if r.skip > 0 {
		buff := make([]byte, r.skip)
		n, err := io.ReadFull(r.inner, buff)
		r.skip -= n
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return 0, err
		}
	}
	return r.inner.Read(p)
}

type readCloser struct {
	io.Reader
	io.Closer
}

type zipEntry struct {
	header zip.FileHeader
	open   func() (io.ReadCloser, error)
}

func (f *zipEntry) FullName() string             { return path.Clean(f.header.Name) }
func (f *zipEntry) Name() string                 { return path.Base(f.FullName()) }
func (f *zipEntry) Size() int64                  { return f.header.FileInfo().Size() }
func (f *zipEntry) Mode() fs.FileMode            { return f.header.FileInfo().Mode() }
func (f *zipEntry) ModTime() time.Time           { return f.header.FileInfo().ModTime() }
func (f *zipEntry) IsDir() bool                  { return f.header.FileInfo().IsDir() }
func (f *zipEntry) Sys() interface{}             { return f.header.FileInfo().Sys() }
func (f *zipEntry) Open() (io.ReadCloser, error) { return f.open() }

func rename(vfs map[string]fsutil.FileReference, oldpath, newpath string) error {
	ref, ok := vfs[oldpath]
	if !ok {
		return &os.LinkError{
			Op:  "rename",
			Old: oldpath,
			New: newpath,
			Err: os.ErrNotExist,
		}
	}
	ref.(*zipEntry).header.Name = newpath
	delete(vfs, oldpath)
	vfs[newpath] = ref
	return nil
}

func create(vfs map[string]fsutil.FileReference, name string, content *zipEntry) {
	content.header.Name = name
	vfs[name] = content
}
