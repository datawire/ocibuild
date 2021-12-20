package pypa_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/dir"
	"github.com/datawire/ocibuild/pkg/python"
	"github.com/datawire/ocibuild/pkg/python/pep376"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
	"github.com/datawire/ocibuild/pkg/python/pypa/direct_url"
	"github.com/datawire/ocibuild/pkg/python/pypa/entry_points"
	"github.com/datawire/ocibuild/pkg/python/pypa/recording_installs"
	"github.com/datawire/ocibuild/pkg/testutil"
)

func pipInstall(ctx context.Context, wheelFile, destDir string) (scheme python.Scheme, err error) {
	maybeSetErr := func(_err error) {
		if _err != nil && err == nil {
			err = _err
		}
	}

	// Step 1: Create the venv
	if err := dexec.CommandContext(ctx, "python3", "-m", "venv", destDir).Run(); err != nil {
		return python.Scheme{}, err
	}
	schemeBytes, err := dexec.CommandContext(ctx, filepath.Join(destDir, "bin", "python3"), "-c", `
import json
from pip._internal.locations import get_scheme;
scheme=get_scheme("")
print(json.dumps({slot: getattr(scheme, slot) for slot in scheme.__slots__}))
`).Output()
	if err != nil {
		return python.Scheme{}, err
	}
	if err := json.Unmarshal(schemeBytes, &scheme); err != nil {
		return python.Scheme{}, err
	}

	if err := os.Rename(destDir, destDir+".lower"); err != nil {
		_ = os.RemoveAll(destDir)
		return python.Scheme{}, err
	}
	defer func() {
		maybeSetErr(os.RemoveAll(destDir + ".lower"))
	}()

	// Step 2: Create the workdir
	if err := os.Mkdir(destDir+".work", 0777); err != nil {
		return python.Scheme{}, err
	}
	defer func() {
		maybeSetErr(os.RemoveAll(destDir + ".work"))
	}()

	// Step 3: Shuffle around "{destDir}" and "{destDir}.upper".
	if err := os.Mkdir(destDir+".upper", 0777); err != nil {
		return python.Scheme{}, err
	}
	if err := os.Mkdir(destDir, 0777); err != nil {
		_ = os.RemoveAll(destDir + ".upper")
		return python.Scheme{}, err
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
		return python.Scheme{}, err
	}
	defer func() {
		maybeSetErr(dexec.CommandContext(ctx, "sudo", "umount", destDir).Run())
		maybeSetErr(os.Remove(destDir))
		maybeSetErr(os.Rename(destDir+".upper", destDir))
	}()

	// Step 4: Actually run pip
	err = dexec.CommandContext(ctx, filepath.Join(destDir, "bin", "pip"), "install", "--no-deps", wheelFile).Run()
	if err != nil {
		return python.Scheme{}, err
	}

	return scheme, nil
}

// Test against the Package Installer for Python.
func TestPIP(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.SkipNow()
	}

	testDownloadedWheels(t, func(t *testing.T, filename string, content []byte) {
		ctx := dlog.NewTestContext(t, true)
		tmpdir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tmpdir, filename), content, 0644))

		// pip reference install
		scheme, err := pipInstall(ctx,
			filepath.Join(tmpdir, filename), // wheelfile
			filepath.Join(tmpdir, "dst"))    // dest dir
		require.NoError(t, err)
		prefix, err := filepath.Rel("/", filepath.Join(tmpdir, "dst"))
		require.NoError(t, err)
		prefix = filepath.ToSlash(prefix)
		expLayer, err := dir.LayerFromDir(filepath.Join(tmpdir, "dst"), prefix)
		require.NoError(t, err)

		// build platform data based on what pip did
		compiler, err := python.ExternalCompiler("python3", "-m", "compileall")
		require.NoError(t, err)
		usr, err := user.Current()
		require.NoError(t, err)
		grp, err := user.LookupGroupId(fmt.Sprintf("%v", os.Getgid()))
		require.NoError(t, err)
		plat := python.Platform{
			ConsoleShebang:   filepath.Join(scheme.Scripts, "python3"),
			GraphicalShebang: filepath.Join(scheme.Scripts, "python3"),
			Scheme:           scheme,
			UID:              os.Getuid(),
			GID:              os.Getgid(),
			UName:            usr.Username,
			GName:            grp.Name,
			PyCompile:        compiler,
		}

		// our own install
		actLayer, err := bdist.InstallWheel(ctx, plat, filepath.Join(tmpdir, filename), bdist.PostInstallHooks(
			pep376.RecordRequested(""),
			entry_points.CreateScripts(plat),
			recording_installs.Record(
				"sha256",
				"pip",
				&direct_url.DirectURL{
					URL:         "file://" + filepath.ToSlash(filepath.Join(tmpdir, filename)),
					ArchiveInfo: &direct_url.ArchiveInfo{},
				},
			),
		))
		require.NoError(t, err)

		// compare them
		testutil.AssertEqualLayers(t, expLayer, actLayer)
	})
}
