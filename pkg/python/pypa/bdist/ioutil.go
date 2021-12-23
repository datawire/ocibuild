package bdist

import (
	"archive/tar"
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python"
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
	isDir := ref.IsDir()
	ref.(*zipEntry).header.Name = newpath
	if isDir {
		ref.(*zipEntry).header.Name += "/"
	}
	delete(vfs, oldpath)
	vfs[newpath] = ref
	return nil
}

func create(vfs map[string]fsutil.FileReference, mtime time.Time, name string, content *zipEntry) {
	isDir := strings.HasSuffix(content.header.Name, "/")
	content.header.Name = name
	if isDir {
		content.header.Name += "/"
	}

	// Discard all permission info except the "execute" bit.
	var externalAttrs python.ZIPExternalAttributes
	switch {
	case isDir:
		externalAttrs.UNIX = python.ModeFmtDir | 0o755
	case isExecutable(content.header):
		externalAttrs.UNIX = python.ModeFmtRegular | 0o755
	default:
		externalAttrs.UNIX = python.ModeFmtRegular | 0o644
	}
	content.header.CreatorVersion = 3 << 8 // force Creator=UNIX
	content.header.ExternalAttrs = externalAttrs.Raw()

	if !mtime.IsZero() {
		// this kills me, but it reflects what `pip` does
		content.header.Modified = mtime
	}

	vfs[name] = content
}

type tarEntry struct {
	header *tar.Header
	open   func() (io.ReadCloser, error)
}

func (f *tarEntry) FullName() string             { return path.Clean(f.header.Name) }
func (f *tarEntry) Name() string                 { return path.Base(f.FullName()) }
func (f *tarEntry) Size() int64                  { return f.header.FileInfo().Size() }
func (f *tarEntry) Mode() fs.FileMode            { return f.header.FileInfo().Mode() }
func (f *tarEntry) ModTime() time.Time           { return f.header.FileInfo().ModTime() }
func (f *tarEntry) IsDir() bool                  { return f.header.FileInfo().IsDir() }
func (f *tarEntry) Sys() interface{}             { return f.header }
func (f *tarEntry) Open() (io.ReadCloser, error) { return f.open() }

func newTarEntry(in fsutil.FileReference, fn func(*tar.Header)) (fsutil.FileReference, error) {
	header, err := tar.FileInfoHeader(in, "")
	if err != nil {
		return nil, err
	}
	header.Name = in.FullName()
	fn(header)
	return &tarEntry{
		header: header,
		open:   in.Open,
	}, nil
}
