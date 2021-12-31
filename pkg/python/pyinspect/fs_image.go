package pyinspect

import (
	"archive/tar"
	"io/fs"
	"path"
	"strings"
	"sync"

	"github.com/datawire/dlib/dexec"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/datawire/ocibuild/pkg/squash"
)

type ImageFS struct {
	Image ociv1.Image

	initOnce sync.Once
	initErr  error
	imgWD    string
	imgPATH  []string
	imgFS    fs.FS
}

var _ FS = (*ImageFS)(nil)

// linuxFilepathSplitList mimics path/filepath.SplitList, but always behaves as if GOOS=linux.
func linuxFilepathSplitList(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, ":")
}

// lookupEnv mimics os.LookupEnv, but always behaves as if GOOS=linux, and operates on a a given
// list of strings rather than os.Environ().  Also there's no caching or anything like that.
func lookupEnv(environ []string, key string) (val string, ok bool) {
	prefix := key + "="
	for _, keyval := range environ {
		if strings.HasPrefix(keyval, prefix) {
			// Mimicking syscall.Getenv() (syscall/env_unix.go), first-match wins.
			return strings.TrimPrefix(keyval, prefix), true
		}
	}
	return "", false
}

func (*ImageFS) Split(pathname string) (dir, file string) { return path.Split(pathname) }
func (*ImageFS) Join(elem ...string) string {
	return path.Join(elem...)
}

func (sys *ImageFS) ensureInitialized() error {
	sys.initOnce.Do(func() {
		sys.initErr = func() error {
			sys.imgWD = "/"
			sys.imgPATH = []string{}

			cfgFile, err := sys.Image.ConfigFile()
			if err != nil {
				return err
			}
			if cfgFile != nil {
				if cfgFile.Config.WorkingDir != "" {
					sys.imgWD = cfgFile.Config.WorkingDir
					if !strings.HasPrefix(sys.imgWD, "/") {
						sys.imgWD = "/" + sys.imgWD
					}
				}
				if _path, ok := lookupEnv(cfgFile.Config.Env, "PATH"); ok {
					sys.imgPATH = linuxFilepathSplitList(_path)
				}
			}

			layers, err := sys.Image.Layers()
			if err != nil {
				return err
			}
			vfs, err := squash.Load(layers, true)
			if err != nil {
				return err
			}
			sys.imgFS = vfs

			return nil
		}()
	})
	return sys.initErr
}

func (sys *ImageFS) Stat(name string) (FileInfo, error) {
	if !path.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}
	if err := sys.ensureInitialized(); err != nil {
		return nil, err
	}
	fileinfo, err := fs.Stat(sys.imgFS, name[1:])
	if err != nil {
		return nil, err
	}
	raw := fileinfo.Sys().(*tar.Header) //nolint:forcetypeassert // if not, this is a bug and it should crash
	return &fileInfo{
		FileInfo: fileinfo,
		uid:      raw.Uid,
		gid:      raw.Gid,
		uname:    raw.Uname,
		gname:    raw.Gname,
	}, nil
}

func (sys *ImageFS) checkExecutable(fullfilename string) error {
	fileinfo, err := sys.Stat(fullfilename)
	if err != nil {
		return err
	}
	if mode := fileinfo.Mode(); mode.IsDir() || mode&0o111 == 0 {
		return fs.ErrPermission
	}
	return nil
}

func (sys *ImageFS) LookPath(filename string) (_ string, err error) {
	defer func() {
		if err != nil {
			err = &fs.PathError{
				Op:   "lookpath",
				Path: filename,
				Err:  err,
			}
		}
	}()

	if err := sys.ensureInitialized(); err != nil {
		return "", err
	}

	if strings.Contains(filename, "/") {
		fullfilename := filename
		if !path.IsAbs(fullfilename) {
			fullfilename = sys.Join(sys.imgWD, fullfilename)
		}
		if err := sys.checkExecutable(fullfilename); err != nil {
			return "", err
		}
		return fullfilename, nil
	}
	for _, dir := range sys.imgPATH {
		fullfilename := sys.Join(dir, filename)
		if !path.IsAbs(fullfilename) {
			fullfilename = sys.Join(sys.imgWD, fullfilename)
		}
		if err := sys.checkExecutable(fullfilename); err == nil {
			return fullfilename, nil
		}
	}
	return "", dexec.ErrNotFound
}
