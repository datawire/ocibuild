package squash

import (
	"archive/tar"
	"io/fs"
	"path"
	"sort"
	"strings"
	"syscall"
)

var (
	ErrLoop   = syscall.ELOOP
	ErrNotDir = syscall.ENOTDIR
)

type fsfile struct {
	name     string // io/fs fullname
	parent   *fsfile
	children map[string]*fsfile

	// if header is nil, that implies that this is a directory
	header *tar.Header
	body   []byte
}

func fsGet(dir *fsfile, pathname string, create, followLinks bool) (*fsfile, error) {
	pathname = path.Clean(pathname)

	done := 0 // index of the next byte in pathname to look at

	// handle absolute paths
	if path.IsAbs(pathname) {
		for dir.parent != dir {
			dir = dir.parent
		}
		done++
	}

	// crawl the tree
	for {
		slash := strings.Index(pathname[done:], "/")
		if slash < 0 {
			break
		}
		var err error
		dir, err = dir.Get(pathname[done:done+slash], create, false)
		done += slash + 1
		if err != nil {
			return nil, err
		}
	}
	ret, err := dir.Get(pathname[done:], create, followLinks)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (f *fsfile) Get(child string, create, followLinks bool) (*fsfile, error) {
	var ret *fsfile

	switch child {
	case "..":
		ret = f.parent
	case ".":
		ret = f
	default:
		if f.header != nil && f.header.Typeflag == tar.TypeSymlink {
			newF, err := fsGet(f.parent, f.header.Linkname, create, true)
			if err != nil {
				return nil, &fs.PathError{
					Op:   "vfs.readlink(dir)",
					Path: "/" + f.name,
					Err:  err,
				}
			}
			f = newF
		}
		// Accessing "foo/bar" implies that "foo" is a directory; if it isn't, then white it
		// out.
		if f.header != nil && f.header.Typeflag != tar.TypeDir {
			if !create {
				return nil, &fs.PathError{
					Op:   "vfs.readdir",
					Path: "/" + f.name,
					Err:  ErrNotDir,
				}
			}
			f.header = nil
			f.body = nil
			whFile, err := f.Get(".wh..wh..opq", true, true)
			if err != nil {
				return nil, err
			}
			if err := whFile.Set(&tar.Header{
				Typeflag: tar.TypeReg,
				Mode:     0o644,
			}, nil); err != nil {
				return nil, err
			}
		}
		// Look up the child
		if f.children == nil {
			f.children = make(map[string]*fsfile)
		}
		if _, ok := f.children[child]; create && !ok {
			f.children[child] = &fsfile{ //nolint:exhaustivestruct
				name:   path.Join(f.name, child),
				parent: f,
			}
		}
		ret = f.children[child]
	}

	// Resolve symlinks
	stack := make(map[*fsfile]struct{})
	for followLinks && ret != nil && ret.header != nil && ret.header.Typeflag == tar.TypeSymlink {
		if _, loop := stack[ret]; loop {
			return nil, &fs.PathError{
				Op:   "vfs.readlink",
				Path: "/" + ret.name,
				Err:  ErrLoop,
			}
		}
		stack[ret] = struct{}{}
		target, err := fsGet(ret.parent, ret.header.Linkname, create, false)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "vfs.readlink",
				Path: "/" + ret.name,
				Err:  err,
			}
		}
		if target == nil {
			break
		}
		ret = target
	}

	// Return
	if ret == nil {
		return nil, &fs.PathError{
			Op:   "vfs.get",
			Path: "/" + path.Join(f.name, child),
			Err:  fs.ErrNotExist,
		}
	}
	return ret, nil
}

func (f *fsfile) Set(hdr *tar.Header, body []byte) error {
	if hdr != nil {
		_hdr := *hdr
		hdr = &_hdr
	}

	wasFile := f.header != nil && f.header.Typeflag != tar.TypeDir

	f.header = hdr
	f.body = body

	if f.header.Typeflag != tar.TypeDir {
		// Changing a directory to a non-directory will implicitly whiteout anything in that
		// directory.
		f.children = nil
	} else if wasFile {
		// If we previously changed a directory to a non-directory, then changing it back
		// risks discarding that implicit whiteout, so if we change a file to a directory,
		// make a potential previous implicit whiteout explicit.  I say "potential" because
		// without the entire layer stack (which this function explicitly doesn't require),
		// we can't know if any given file was such a dir-to-non-dir conversion.
		whFile, err := f.Get(".wh..wh..opq", true, true)
		if err != nil {
			return err
		}
		if err := whFile.Set(&tar.Header{
			Typeflag: tar.TypeReg,
			Mode:     0o644,
		}, nil); err != nil {
			return err
		}
	}
	if f.parent != nil {
		if basename := path.Base(f.name); strings.HasPrefix(basename, ".wh.") {
			if basename == ".wh..wh..opq" {
				for k := range f.parent.children {
					if k != basename {
						delete(f.parent.children, k)
					}
				}
			} else {
				delete(f.parent.children, strings.TrimPrefix(basename, ".wh."))
			}
		} else {
			delete(f.parent.children, ".wh."+basename)
		}
	}
	return nil
}

func (f *fsfile) WriteTo(tarWriter *tar.Writer) error {
	name := f.name

	if f.header != nil {
		if f.header.Typeflag == tar.TypeDir {
			name += "/"
		}
		hdr := *f.header // shallow copy
		hdr.Name = name
		if err := tarWriter.WriteHeader(&hdr); err != nil {
			return err
		}
		if _, err := tarWriter.Write(f.body); err != nil {
			return err
		}
	}

	childNames := make([]string, 0, len(f.children))
	for childName := range f.children {
		childNames = append(childNames, childName)
	}
	sort.Slice(childNames, func(i, j int) bool {
		iStr := childNames[i]
		jStr := childNames[j]
		iWhiteout := strings.HasPrefix(iStr, ".wh.")
		jWhiteout := strings.HasPrefix(jStr, ".wh.")
		switch {
		case iWhiteout && !jWhiteout:
			return true
		case !iWhiteout && jWhiteout:
			return false
		}
		return iStr < jStr
	})

	for _, childName := range childNames {
		child := f.children[childName]
		if err := child.WriteTo(tarWriter); err != nil {
			return err
		}
	}

	return nil
}
