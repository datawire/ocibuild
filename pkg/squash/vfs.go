package squash

import (
	"archive/tar"
	"path"
	"sort"
	"strings"
)

type fsfile struct {
	name     string
	parent   *fsfile
	children map[string]*fsfile

	header *tar.Header
	body   []byte
}

func fsGet(dir *fsfile, pathname string) *fsfile {
	pathname = path.Clean(pathname)

	// handle absolute paths
	if path.IsAbs(pathname) {
		for dir.parent != nil {
			dir = dir.parent
		}
		pathname = pathname[1:]
	}

	// crawl the tree
	for {
		slash := strings.Index(pathname, "/")
		if slash < 0 {
			break
		}
		dir = dir.Get(pathname[:slash])
		if dir == nil {
			return nil
		}
		pathname = pathname[slash+1:]
	}
	return dir.Get(pathname)
}

func (f *fsfile) Get(child string) *fsfile {
	var ret *fsfile

	switch child {
	case "..":
		ret = f.parent
	case ".":
		ret = f
	default:
		// Accessing "foo/bar" implies that "foo" is a directory; if it isn't, then white it
		// out.
		if f.header != nil && f.header.Typeflag != tar.TypeDir {
			f.header = nil
			f.body = nil
			f.Get(".wh..wh..opq").Set(&tar.Header{
				Typeflag: tar.TypeReg,
				Mode:     0o644,
			}, nil)
		}
		// Look up the child
		if f.children == nil {
			f.children = make(map[string]*fsfile)
		}
		if _, ok := f.children[child]; !ok {
			f.children[child] = &fsfile{
				name:   child,
				parent: f,
			}
		}
		ret = f.children[child]
	}

	// Resolve symlinks
	for ret != nil && ret.header != nil && ret.header.Typeflag == tar.TypeSymlink {
		target := fsGet(f, ret.header.Linkname)
		if target == nil {
			break
		}
		ret = target
	}

	// Return
	return ret
}

func (f *fsfile) Set(hdr *tar.Header, body []byte) {
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
		f.Get(".wh..wh..opq").Set(&tar.Header{
			Typeflag: tar.TypeReg,
			Mode:     0o644,
		}, nil)
	}
	if f.parent != nil {
		if strings.HasPrefix(f.name, ".wh.") {
			if f.name == ".wh..wh..opq" {
				for k := range f.parent.children {
					if k != f.name {
						delete(f.parent.children, k)
					}
				}
			} else {
				delete(f.parent.children, strings.TrimPrefix(f.name, ".wh."))
			}
		} else {
			delete(f.parent.children, ".wh."+f.name)
		}
	}
}

func (f *fsfile) WriteTo(basedir string, tarWriter *tar.Writer) error {
	name := path.Join(basedir, f.name)

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
		if err := child.WriteTo(name, tarWriter); err != nil {
			return err
		}
	}

	return nil
}
