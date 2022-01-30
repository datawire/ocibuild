package dockerutil

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/datawire/dlib/dexec"
	"github.com/google/go-containerregistry/pkg/name"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

func newTag(repo string) (name.Tag, error) {
	return name.NewTag(fmt.Sprintf("ocibuild.local/%s:%d.%d",
		repo, os.Getpid(), time.Now().UnixNano()))
}

func WithImage(
	ctx context.Context,
	imgname string,
	img ociv1.Image,
	fn func(context.Context, name.Tag) error,
) (err error) {
	maybeSetErr := func(_err error) {
		if _err != nil && err == nil {
			err = _err
		}
	}

	tag, err := newTag(imgname)
	if err != nil {
		return err
	}
	defer func() {
		maybeSetErr(dexec.CommandContext(ctx, "docker", "image", "rm", tag.String()).Run())
	}()
	cmd := dexec.CommandContext(ctx, "docker", "image", "load")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		_ = pipe.Close()
		_ = cmd.Wait()
	}()
	if err := ociv1tarball.Write(tag, img, pipe); err != nil {
		return err
	}
	if err := pipe.Close(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return fn(ctx, tag)
}
