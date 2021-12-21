package pypa_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/dir"
	"github.com/datawire/ocibuild/pkg/python"
	"github.com/datawire/ocibuild/pkg/python/pep376"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
	"github.com/datawire/ocibuild/pkg/python/pypa/direct_url"
	"github.com/datawire/ocibuild/pkg/python/pypa/entry_points"
	"github.com/datawire/ocibuild/pkg/python/pypa/recording_installs"
	"github.com/datawire/ocibuild/pkg/reproducible"
	"github.com/datawire/ocibuild/pkg/testutil"
)

func pipInstall(ctx context.Context, destDir, wheelFile string) (ociv1.Layer, error) {
	if err := os.MkdirAll(destDir, 0777); err != nil {
		return nil, err
	}

	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	grp, err := user.LookupGroupId(fmt.Sprintf("%v", os.Getgid()))
	if err != nil {
		return nil, err
	}

	cmd := dexec.CommandContext(ctx, "pip3", "install", "--no-deps", "--prefix="+destDir, wheelFile)
	cmd.Env = append(os.Environ(),
		"PYTHONHASHSEED=0",
		fmt.Sprintf("SOURCE_DATE_EPOCH=%d", reproducible.Now().Unix()))
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	layerPrefix, err := filepath.Rel("/", destDir)
	if err != nil {
		return nil, err
	}
	layerPrefix = filepath.ToSlash(layerPrefix)
	return dir.LayerFromDir(
		destDir,
		&dir.Prefix{
			DirName: layerPrefix,
			UID:     os.Getuid(),
			GID:     os.Getgid(),
			UName:   usr.Username,
			GName:   grp.Name,
		},
		reproducible.Now(),
	)
}

// Return a python.Platform that mimics the behavior of `pip3 install --prefix=${destDir}`.
func pipPlatform(ctx context.Context, destDir string) (python.Platform, error) {

	// 1. Look up user info.
	usr, err := user.Current()
	if err != nil {
		return python.Platform{}, err
	}
	grp, err := user.LookupGroupId(fmt.Sprintf("%v", os.Getgid()))
	if err != nil {
		return python.Platform{}, err
	}

	// 2. Look up the scheme.
	schemeBytes, err := dexec.CommandContext(ctx, "python3", "-c", `
import sys
import json
from pip._internal.locations import get_scheme;
scheme=get_scheme("", prefix=sys.argv[1])
print(json.dumps({slot: getattr(scheme, slot) for slot in scheme.__slots__}))
`, destDir).Output()
	if err != nil {
		return python.Platform{}, err
	}
	var scheme python.Scheme
	if err := json.Unmarshal(schemeBytes, &scheme); err != nil {
		return python.Platform{}, err
	}

	// 3. Look up pip3's shebang.
	pip3path, err := dexec.LookPath("pip3")
	if err != nil {
		return python.Platform{}, err
	}
	pip3bytes, err := os.ReadFile(pip3path)
	if err != nil {
		return python.Platform{}, err
	}
	pip3shebang := strings.TrimSpace(strings.TrimPrefix(string(bytes.SplitN(pip3bytes, []byte("\n"), 2)[0]), "#!"))

	// 4. Assemble the compiler.
	compiler, err := python.ExternalCompiler("python3", "-m", "compileall")
	if err != nil {
		return python.Platform{}, err
	}

	// 5. Put it all together.
	return python.Platform{
		ConsoleShebang:   pip3shebang,
		GraphicalShebang: pip3shebang,
		Scheme:           scheme,
		UID:              os.Getuid(),
		GID:              os.Getgid(),
		UName:            usr.Username,
		GName:            grp.Name,
		PyCompile:        compiler,
	}, nil
}

// Test against the Package Installer for Python.
func TestPIP(t *testing.T) {
	t.Logf("reproducible.Now() => %v", reproducible.Now())

	testDownloadedWheels(t, func(t *testing.T, filename string, content []byte) {
		ctx := dlog.NewTestContext(t, true)
		//tmpdir := t.TempDir()
		tmpdir := "/tmp/x"
		os.RemoveAll(tmpdir)
		os.Mkdir(tmpdir, 0755)

		require.NoError(t, os.WriteFile(filepath.Join(tmpdir, filename), content, 0644))

		// pip reference install
		expLayer, err := pipInstall(ctx,
			filepath.Join(tmpdir, "dst"),    // dest dir
			filepath.Join(tmpdir, filename)) // wheelfile
		require.NoError(t, err)

		// build platform data to mimic what pip did
		plat, err := pipPlatform(ctx, filepath.Join(tmpdir, "dst"))
		require.NoError(t, err)

		// our own install
		actLayer, err := bdist.InstallWheel(ctx,
			plat,
			reproducible.Now(), // minTime
			reproducible.Now(), // maxTime
			filepath.Join(tmpdir, filename),
			bdist.PostInstallHooks(
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
			),
		)
		require.NoError(t, err)

		// compare them
		testutil.AssertEqualLayers(t, expLayer, actLayer)
	})
}
