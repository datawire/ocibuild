package pep427_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/dir"
	_ "github.com/datawire/ocibuild/pkg/pep427"
)

func pipInstall(ctx context.Context, wheelFile, destDir string) (err error) {
	maybeSetErr := func(_err error) {
		if _err != nil && err == nil {
			err = _err
		}
	}

	// Step 1: Create the venv
	if err := dexec.CommandContext(ctx, "python3", "-m", "venv", destDir).Run(); err != nil {
		return err
	}
	if err := os.Rename(destDir, destDir+".lower"); err != nil {
		_ = os.RemoveAll(destDir)
		return err
	}
	defer func() {
		maybeSetErr(os.RemoveAll(destDir + ".lower"))
	}()

	// Step 2: Create the workdir
	if err := os.Mkdir(destDir+".work", 0777); err != nil {
		return err
	}
	defer func() {
		maybeSetErr(os.RemoveAll(destDir + ".work"))
	}()

	// Step 3: Shuffle around "{destDir}" and "{destDir}.upper".
	if err := os.Mkdir(destDir+".upper", 0777); err != nil {
		return err
	}
	if err := os.Mkdir(destDir, 0777); err != nil {
		maybeSetErr(os.RemoveAll(destDir + ".upper"))
		return err
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
		return err
	}
	defer func() {
		maybeSetErr(dexec.CommandContext(ctx, "sudo", "umount", destDir).Run())
		maybeSetErr(os.Remove(destDir))
		maybeSetErr(os.Rename(destDir+".upper", destDir))
	}()

	// Step 4: Actually run pip
	err = dexec.CommandContext(ctx, filepath.Join(destDir, "bin", "pip"), "install", wheelFile).Run()
	if err != nil {
		return err
	}

	return nil
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
			require.NoError(t, pipInstall(ctx,
				filepath.Join("testdata", name), // wheelfile
				filepath.Join(tmpdir, "dst")))   // dest dir
			expLayer, err := dir.LayerFromDir(tmpdir)
			require.NoError(t, err)

			// TODO: run pkg/pep427 (set Platform dirs to match {dir}/dst)
			actLayer := ociv1.Layer(nil)

			assert.Equal(t, expLayer, actLayer)
		})
	}
}
