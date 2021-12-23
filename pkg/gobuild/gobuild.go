// Package gobuild deals with creating a layer of Go binaries.
package gobuild

import (
	"context"
	"os"
	"time"

	"github.com/datawire/dlib/dexec"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"

	"github.com/datawire/ocibuild/pkg/dir"
)

func LayerFromGo(ctx context.Context, clampTime time.Time, pkgnames []string, opts ...ociv1tarball.LayerOption) (_ ociv1.Layer, err error) {
	maybeSetErr := func(_err error) {
		if _err != nil && err == nil {
			err = _err
		}
	}

	tmpdir, err := os.MkdirTemp("", "ocibuild-gobuild.")
	if err != nil {
		return nil, err
	}
	defer func() {
		maybeSetErr(os.RemoveAll(tmpdir))
	}()

	// TODO(lukeshu): Call or mimic code from Ko in order to figure out multi-arch support.
	args := append([]string{
		"go", "build",
		"-trimpath",
		"-o", tmpdir,
		"--",
	}, pkgnames...)
	cmd := dexec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64")

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return dir.LayerFromDir(tmpdir, &dir.Prefix{
		DirName: "usr/local/bin",
		UName:   "root",
		GName:   "root",
	}, clampTime, opts...)
}
