package squash_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	"github.com/google/go-containerregistry/pkg/name"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/dockerutil"
	"github.com/datawire/ocibuild/pkg/squash"
)

type TestFile struct {
	Name     string
	Type     byte
	Linkname string

	NoDocker   bool
	NoOCIBuild bool
}

type TestLayer []TestFile

func ParseTestLayer(t *testing.T, layer ociv1.Layer) TestLayer {
	t.Helper()
	var ret TestLayer

	layerReader, err := layer.Uncompressed()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		assert.NoError(t, layerReader.Close())
	}()

	tarReader := tar.NewReader(layerReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}
		_, err = io.ReadAll(tarReader)
		if err != nil {
			t.Fatal(err)
		}

		ret = append(ret, TestFile{ //nolint:exhaustivestruct
			Name:     header.Name,
			Type:     header.Typeflag,
			Linkname: header.Linkname,
		})
	}

	return ret
}

func (tl TestLayer) ToLayer(t *testing.T) ociv1.Layer {
	t.Helper()
	var byteWriter bytes.Buffer
	tarWriter := tar.NewWriter(&byteWriter)
	for _, file := range tl {
		header := &tar.Header{
			Name:     file.Name,
			Typeflag: file.Type,
			Linkname: file.Linkname,
			Size:     0,
			Mode:     0o644,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}

	// Wrap that in to a Layer object
	byteSlice := byteWriter.Bytes()
	ret, err := ociv1tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(byteSlice)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return ret
}

func TestSquash(t *testing.T) {
	t.Parallel()

	//nolint:lll // big table
	testcases := map[string]struct {
		Input  []TestLayer
		Output TestLayer
	}{
		"sanitize": {
			Input: []TestLayer{
				{
					{Name: ".", Type: tar.TypeDir},                     // add trailing "/"
					{Name: "./foo/bar", Type: tar.TypeReg},             // trim leading "./"
					{Name: "foo/./baz", Type: tar.TypeReg},             // clean path
					{Name: "foo/.aaa", Type: tar.TypeReg},              // sorted *after* the whiteout
					{Name: "foo/../foo/.wh.qux", Type: tar.TypeReg},    // clean path; sorted before .aaa
					{Name: "foo", Type: tar.TypeDir},                   // add trailing "/"
					{Name: "foo/bar/sub/../../zap", Type: tar.TypeReg}, // don't imply that bar is a dir
					{Name: "foo/d/x", Type: tar.TypeReg},               // masked by non-dir foo/d"
					{Name: "foo/d", Type: tar.TypeReg},                 // masks "foo/d/*"
					{Name: "sym", Type: tar.TypeSymlink, Linkname: "foo"},
					{Name: "sym/moved", Type: tar.TypeReg}, // symlink resolved to "foo/moved"
				},
			},
			Output: TestLayer{
				{Name: "./", Type: tar.TypeDir, NoDocker: true},
				{Name: "foo/", Type: tar.TypeDir},
				{Name: "foo/.wh.qux", Type: tar.TypeReg, NoDocker: true},
				{Name: "foo/.aaa", Type: tar.TypeReg},
				{Name: "foo/bar", Type: tar.TypeReg},
				{Name: "foo/baz", Type: tar.TypeReg},
				{Name: "foo/d", Type: tar.TypeReg},
				{Name: "foo/moved", Type: tar.TypeReg},
				{Name: "foo/zap", Type: tar.TypeReg},
				{Name: "sym", Type: tar.TypeSymlink, Linkname: "foo"},
			},
		},
		"opaque": {
			Input: []TestLayer{
				{
					{Name: "dir/foo", Type: tar.TypeReg},
				},
				{
					{Name: "dir/bar", Type: tar.TypeReg},
					{Name: "dir/.wh..wh..opq", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "dir/", Type: tar.TypeDir, NoOCIBuild: true},
				{Name: "dir/.wh..wh..opq", Type: tar.TypeReg, NoDocker: true},
				{Name: "dir/bar", Type: tar.TypeReg},
			},
		},
		"opaque-implicit-1": {
			Input: []TestLayer{
				{
					{Name: "dir/foo", Type: tar.TypeReg},
				},
				{
					{Name: "dir", Type: tar.TypeReg},
				},
				{
					{Name: "dir/bar", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "dir/.wh..wh..opq", Type: tar.TypeReg, NoDocker: true},
				{Name: "dir/bar", Type: tar.TypeReg},
			},
		},
		"opaque-implicit-2": {
			Input: []TestLayer{
				{
					{Name: "dir/foo", Type: tar.TypeReg},
				},
				{
					{Name: "dir", Type: tar.TypeReg},
				},
				{
					{Name: "dir", Type: tar.TypeDir},
				},
			},
			Output: TestLayer{
				{Name: "dir/", Type: tar.TypeDir},
				{Name: "dir/.wh..wh..opq", Type: tar.TypeReg, NoDocker: true},
			},
		},
		"symlink-to-file": {
			Input: []TestLayer{
				{
					{Name: "foo", Type: tar.TypeSymlink, Linkname: "bar"},
				},
				{
					{Name: "foo", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "foo", Type: tar.TypeReg},
			},
		},
		"symlink-to-symlink": {
			Input: []TestLayer{
				{
					{Name: "foo", Type: tar.TypeSymlink, Linkname: "bar"},
				},
				{
					{Name: "foo", Type: tar.TypeSymlink, Linkname: "baz"},
				},
			},
			Output: TestLayer{
				{Name: "foo", Type: tar.TypeSymlink, Linkname: "baz"},
			},
		},
		"symlink-in-dir-simple": {
			Input: []TestLayer{
				{
					{Name: "lnkdir", Type: tar.TypeSymlink, Linkname: "tgtdir"},
					{Name: "tgtdir", Type: tar.TypeDir},
					{Name: "lnkdir/file", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "lnkdir", Type: tar.TypeSymlink, Linkname: "tgtdir"},
				{Name: "tgtdir/", Type: tar.TypeDir},
				{Name: "tgtdir/file", Type: tar.TypeReg},
			},
		},
		"symlink-in-dir-dne": {
			Input: []TestLayer{
				{
					{Name: "lnkdir", Type: tar.TypeSymlink, Linkname: "tgtdir"}, // does not exist
					{Name: "lnkdir/file", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "lnkdir", Type: tar.TypeSymlink, Linkname: "tgtdir"},
				{Name: "tgtdir/", Type: tar.TypeDir, NoOCIBuild: true},
				{Name: "tgtdir/file", Type: tar.TypeReg},
			},
		},
		"symlink-in-dir-outside": {
			Input: []TestLayer{
				{
					{Name: "lnkdir", Type: tar.TypeSymlink, Linkname: "../tgtdir"}, // outside of image
					{Name: "tgtdir", Type: tar.TypeDir},
				},
				{
					{Name: "lnkdir/file", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "lnkdir", Type: tar.TypeSymlink, Linkname: "../tgtdir"},
				{Name: "tgtdir/", Type: tar.TypeDir},
				{Name: "tgtdir/file", Type: tar.TypeReg},
			},
		},
		"symlink-in-dir-absolute": {
			Input: []TestLayer{
				{
					{Name: "dir", Type: tar.TypeDir},
					{Name: "dir/lnkdir", Type: tar.TypeSymlink, Linkname: "/tgtdir"}, // absolute
					{Name: "tgtdir", Type: tar.TypeDir},
				},
				{
					{Name: "dir/lnkdir/file", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "dir/", Type: tar.TypeDir},
				{Name: "dir/lnkdir", Type: tar.TypeSymlink, Linkname: "/tgtdir"},
				{Name: "tgtdir/", Type: tar.TypeDir},
				{Name: "tgtdir/file", Type: tar.TypeReg},
			},
		},
	}

	for tcName, tc := range testcases {
		tc := tc
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()

			input := make([]ociv1.Layer, 0, len(tc.Input))
			for _, l := range tc.Input {
				input = append(input, l.ToLayer(t))
			}

			t.Run("ocibuild", func(t *testing.T) { // to test the code
				t.Parallel()

				var expected TestLayer
				for _, file := range tc.Output {
					if file.NoOCIBuild {
						continue
					}
					file.NoDocker = false
					file.NoOCIBuild = false
					expected = append(expected, file)
				}

				actual, err := squash.Squash(input)
				require.NoError(t, err)
				assert.Equal(t, expected, ParseTestLayer(t, actual))
			})
			t.Run("docker", func(t *testing.T) { // to test the testcase itself
				t.Parallel()

				var expected TestLayer
				for _, file := range tc.Output {
					if file.NoDocker {
						continue
					}
					file.NoDocker = false
					file.NoOCIBuild = false
					expected = append(expected, file)
				}

				actual := dockerSquash(t, input)
				assert.Equal(t, expected, actual)
			})
		})
	}
}

func dockerSquash(t *testing.T, layers []ociv1.Layer) TestLayer { //nolint:thelper // useful in trace
	ctx := dlog.NewTestContext(t, true)

	img, err := mutate.AppendLayers(empty.Image, layers...)
	require.NoError(t, err)

	ran := false
	var layer ociv1.Layer
	err = dockerutil.WithImage(ctx, "squash-test", img, func(ctx context.Context, tag name.Tag) (err error) {
		ran = true
		maybeSetErr := func(_err error) {
			if _err != nil && err == nil {
				err = _err
			}
		}
		bs, err := dexec.CommandContext(ctx, "docker", "container", "create", tag.String(), "/bin/sh").Output()
		if err != nil {
			return err
		}
		containerName := strings.TrimSpace(string(bs))
		defer func() {
			maybeSetErr(dexec.CommandContext(ctx, "docker", "container", "rm", containerName).Run())
		}()

		cmd := dexec.CommandContext(ctx, "docker", "container", "export", containerName)
		pipe, err := cmd.StdoutPipe()
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
		layer, err = ociv1tarball.LayerFromReader(pipe)
		if err != nil {
			return err
		}
		if err := pipe.Close(); err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			return err
		}

		return nil
	})
	if !ran {
		t.Log("don't have a functioning docker")
		t.SkipNow()
	}
	require.NoError(t, err)
	var ret TestLayer //nolint:prealloc // 'continue' is likely
	for _, file := range ParseTestLayer(t, layer) {
		if strings.HasPrefix(file.Name, "dev/") ||
			strings.HasPrefix(file.Name, "proc/") ||
			strings.HasPrefix(file.Name, "sys/") ||
			strings.HasPrefix(file.Name, "etc/") ||
			file.Name == ".dockerenv" {
			continue
		}
		ret = append(ret, file)
	}
	return ret
}
