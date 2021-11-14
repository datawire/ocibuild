package pep427_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/dir"
	"github.com/datawire/ocibuild/pkg/pep427"
	"github.com/datawire/ocibuild/pkg/python"
)

func pipInstall(ctx context.Context, wheelFile, destDir string) (scheme pep427.Scheme, err error) {
	maybeSetErr := func(_err error) {
		if _err != nil && err == nil {
			err = _err
		}
	}

	// Step 1: Create the venv
	if err := dexec.CommandContext(ctx, "python3", "-m", "venv", destDir).Run(); err != nil {
		return pep427.Scheme{}, err
	}
	schemeBytes, err := dexec.CommandContext(ctx, filepath.Join(destDir, "bin", "python3"), "-c", `
import json
from pip._internal.locations import get_scheme;
scheme=get_scheme("")
print(json.dumps({slot: getattr(scheme, slot) for slot in scheme.__slots__}))
`).Output()
	if err != nil {
		return pep427.Scheme{}, err
	}
	if err := json.Unmarshal(schemeBytes, &scheme); err != nil {
		return pep427.Scheme{}, err
	}

	if err := os.Rename(destDir, destDir+".lower"); err != nil {
		_ = os.RemoveAll(destDir)
		return pep427.Scheme{}, err
	}
	defer func() {
		maybeSetErr(os.RemoveAll(destDir + ".lower"))
	}()

	// Step 2: Create the workdir
	if err := os.Mkdir(destDir+".work", 0777); err != nil {
		return pep427.Scheme{}, err
	}
	defer func() {
		maybeSetErr(os.RemoveAll(destDir + ".work"))
	}()

	// Step 3: Shuffle around "{destDir}" and "{destDir}.upper".
	if err := os.Mkdir(destDir+".upper", 0777); err != nil {
		return pep427.Scheme{}, err
	}
	if err := os.Mkdir(destDir, 0777); err != nil {
		_ = os.RemoveAll(destDir + ".upper")
		return pep427.Scheme{}, err
	}
	if err := dexec.CommandContext(ctx,
		"sudo", "mount",
		"-t", "overlay", // filesystem type
		"-o", strings.Join([]string{ // filesystem options
			"lowerdir=" + (destDir + ".lower"),
			"upperdir=" + (destDir + ".upper"),
			"workdir=" + (destDir + ".work"),
		}, ","),
		"overlay:"+filepath.Base(wheelFile), // device; for the 'overlay' FS type, this is just a vanity name
		destDir,                             // mountpoint
	).Run(); err != nil {
		maybeSetErr(os.RemoveAll(destDir + ".upper"))
		maybeSetErr(os.RemoveAll(destDir))
		return pep427.Scheme{}, err
	}
	defer func() {
		maybeSetErr(dexec.CommandContext(ctx, "sudo", "umount", destDir).Run())
		maybeSetErr(os.Remove(destDir))
		maybeSetErr(os.Rename(destDir+".upper", destDir))
	}()

	// Step 4: Actually run pip
	err = dexec.CommandContext(ctx, filepath.Join(destDir, "bin", "pip"), "install", wheelFile).Run()
	if err != nil {
		return pep427.Scheme{}, err
	}

	return scheme, nil
}

// Test against the Package Installer for Python.
func TestPIP(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.SkipNow()
	}

	dirents, err := os.ReadDir("testdata")
	require.NoError(t, err)
	for _, dirent := range dirents {
		name := dirent.Name()
		if !(strings.HasSuffix(name, ".whl") && dirent.Type().IsRegular()) {
			continue
		}
		t.Run(name, func(t *testing.T) {
			ctx := dlog.NewTestContext(t, true)
			tmpdir := t.TempDir()

			// pip reference install
			scheme, err := pipInstall(ctx,
				filepath.Join("testdata", name), // wheelfile
				filepath.Join(tmpdir, "dst"))    // dest dir
			require.NoError(t, err)
			expLayer, err := dir.LayerFromDir(tmpdir)
			require.NoError(t, err)

			// build platform data based on what pip did
			compiler, err := python.ExternalCompiler("python3", "-m", "compileall")
			require.NoError(t, err)
			plat := pep427.Platform{
				ConsoleShebang:   filepath.Join(scheme.Scripts, "python3"),
				GraphicalShebang: filepath.Join(scheme.Scripts, "python3"),
				Scheme:           scheme,
				PyCompile:        compiler,
			}

			// our own install
			actLayer, err := pep427.InstallWheel(ctx, plat, filepath.Join("testdata", name))
			require.NoError(t, err)

			// compare them
			assert.Equal(t, expLayer, actLayer)
		})
	}
}
