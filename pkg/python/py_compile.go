package python

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/datawire/dlib/dexec"

	"github.com/datawire/ocibuild/pkg/fsutil"
)

// A Compiler is a function that takes any number of source .py files, and emits any number of
// compiled .pyc files.
//
// The pythonPath argument is a list of `io/fs`-style paths to insert in to PYTHONPATH, so that `in`
// source files can refer to eachother.
//
// The returned output does *not* include directories.  The ordering of the output is undefined.
type Compiler func(ctx context.Context, clampTime time.Time, pythonPath []string, in []fsutil.FileReference) ([]fsutil.FileReference, error)

// ExternalCompiler returns a `Compiler` that uses an external command to compile .py files to .pyc
// files.  It is designed for use with Python's "compileall" module.  It makes use of the "-p" flag
// and passes a directory rather than a single file; so the "py_compile" module is not appropriate.
//
// For example:
//
//     plat.Compile = ExternalCompiler("python3", "-m", "compileall")
func ExternalCompiler(cmdline ...string) (Compiler, error) {
	exe, err := dexec.LookPath(cmdline[0])
	if err != nil {
		return nil, err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, clampTime time.Time, pythonPath []string, in []fsutil.FileReference) (_ []fsutil.FileReference, err error) {
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

		writeFile := func(inFile fsutil.FileReference) (err error) {
			maybeSetErr := func(_err error) {
				if _err != nil && err == nil {
					err = _err
				}
			}

			tmpfilename := filepath.Join(tmpdir, filepath.FromSlash(inFile.FullName()))

			if err := os.MkdirAll(filepath.Dir(tmpfilename), 0o777); err != nil {
				return err
			}

			// File content
			outWriter, err := os.Create(tmpfilename)
			if err != nil {
				return err
			}
			defer func() {
				if outWriter != nil {
					maybeSetErr(outWriter.Close())
				}
			}()
			inReader, err := inFile.Open()
			if err != nil {
				return err
			}
			defer func() {
				if inReader != nil {
					maybeSetErr(inReader.Close())
				}
			}()
			if _, err := io.Copy(outWriter, inReader); err != nil {
				return err
			}
			if err := outWriter.Close(); err != nil {
				return err
			}
			outWriter = nil
			if err := inReader.Close(); err != nil {
				return err
			}
			inReader = nil

			// File metadata
			if err := os.Chtimes(tmpfilename, inFile.ModTime(), inFile.ModTime()); err != nil {
				return err
			}

			return nil
		}

		for _, inFile := range in {
			if err := writeFile(inFile); err != nil {
				return nil, err
			}
		}

		// Run the compiler
		cmd := dexec.CommandContext(ctx, exe, append(cmdline[1:],
			"-s", tmpdir, // strip-dir for the in-.pyc filename
			"-p", "/", // prepend-dir for the in-.pyc filename
			tmpdir, // directory to compile
		)...)

		cmd.Env = append(os.Environ(),
			"PYTHONHASHSEED=0")
		if len(pythonPath) > 0 {
			var pythonPathEnv []string
			for _, dir := range pythonPath {
				pythonPathEnv = append(pythonPathEnv, filepath.Join(tmpdir, filepath.FromSlash(dir)))
			}
			if e := os.Getenv("PYTHONPATH"); e != "" {
				pythonPathEnv = append(pythonPathEnv, e)
			}
			cmd.Env = append(cmd.Env,
				"PYTHONPATH="+strings.Join(pythonPathEnv, string(filepath.ListSeparator)))
		}
		if !clampTime.IsZero() {
			cmd.Env = append(cmd.Env,
				fmt.Sprintf("SOURCE_DATE_EPOCH=%d", clampTime.Unix()))
		}

		if err := cmd.Run(); err != nil {
			return nil, err
		}

		// Read in the output
		var ret []fsutil.FileReference
		// vfs["slash-path"] and zipEntry.Name are slash-paths, so use fs.WalkDir instead of
		// filepath.Walk so that we don't need to worry about converting between forward and
		// backward slashes.
		dirFS := os.DirFS(tmpdir)
		err = fs.WalkDir(dirFS, ".", func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() || !strings.HasSuffix(p, ".pyc") {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			var content []byte
			fh, err := dirFS.Open(p)
			if err != nil {
				return err
			}
			defer func() {
				_ = fh.Close()
			}()
			content, err = io.ReadAll(fh)
			if err != nil {
				return err
			}
			ret = append(ret, &fsutil.InMemFileReference{
				FileInfo:  info,
				MFullName: p,
				MContent:  content,
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
		return ret, nil
	}, nil
}
