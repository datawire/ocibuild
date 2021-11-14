package pep427

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/datawire/dlib/dexec"
)

type PyCompiler func(context.Context, *zipEntry) (map[string]*zipEntry, error)

// plat.PyCompile = ExternalPyCompiler("python3", "-m", "compileall")
func ExternalPyCompiler(cmdline ...string) (PyCompiler, error) {
	exe, err := dexec.LookPath(cmdline[0])
	if err != nil {
		return nil, err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, in *zipEntry) (compiled map[string]*zipEntry, err error) {
		maybeSetErr := func(_err error) {
			if _err != nil && err == nil {
				err = _err
			}
		}

		// Set up the tmpdir
		tmpdir, err := os.MkdirTemp("", "ocibuild-pycompile.")
		if err != nil {
			return nil, err
		}
		defer func() {
			maybeSetErr(os.RemoveAll(tmpdir))
		}()

		// Get the input file
		inReader, err := in.Open()
		if err != nil {
			return nil, err
		}
		inBytes, err := io.ReadAll(inReader)
		if err != nil {
			_ = inReader.Close()
			return nil, err
		}
		if err := inReader.Close(); err != nil {
			return nil, err
		}

		// Write the input file to the tempdir
		filename := filepath.Join(tmpdir, path.Base(in.Name))
		if err := os.WriteFile(filename, inBytes, 0666); err != nil {
			return nil, err
		}
		if err := os.Chtimes(filename, in.Modified, in.Modified); err != nil {
			return nil, err
		}

		// Run the compiler
		cmd := dexec.CommandContext(ctx, exe, append(cmdline[1:],
			"-p", path.Dir(in.Name), // prepend-dir for the in-.pyc filename
			path.Base(in.Name), // file to compile
		)...)
		cmd.Dir = tmpdir
		if err := cmd.Run(); err != nil {
			return nil, err
		}

		// Read in the output
		vfs := make(map[string]*zipEntry)
		// vfs["slash-path"] and zipEntry.Name are slash-paths, so use fs.WalkDir instead of
		// filepath.Walk so that we don't need to worry about converting between forward and
		// backward slashes.
		dirFS := os.DirFS(tmpdir)
		err = fs.WalkDir(dirFS, "/", func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() || !strings.HasSuffix(p, ".py") {
				return nil
			}
			fh, err := dirFS.Open(p)
			if err != nil {
				return err
			}
			defer func() {
				_ = fh.Close()
			}()
			content, err := io.ReadAll(fh)
			if err != nil {
				return err
			}
			vfs[p] = &zipEntry{
				FileHeader: zip.FileHeader{
					Name:     p,
					Modified: in.Modified,
				},
				Open: func() (io.ReadCloser, error) {
					return io.NopCloser(bytes.NewReader(content)), nil
				},
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return vfs, nil
	}, nil
}
