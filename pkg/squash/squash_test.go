package squash

import (
	"archive/tar"
	"bytes"
	"io"
	"testing"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/stretchr/testify/assert"
)

type TestFile struct {
	Name     string
	Type     byte
	Linkname string
}

type TestLayer []TestFile

func ParseTestLayer(t *testing.T, layer ociv1.Layer) TestLayer {
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
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		_, err = io.ReadAll(tarReader)
		if err != nil {
			t.Fatal(err)
		}

		ret = append(ret, TestFile{
			Name:     header.Name,
			Type:     header.Typeflag,
			Linkname: header.Linkname,
		})
	}

	return ret
}

func (tl TestLayer) ToLayer(t *testing.T) ociv1.Layer {
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
				{Name: "./", Type: tar.TypeDir},
				{Name: "foo/", Type: tar.TypeDir},
				{Name: "foo/.wh.qux", Type: tar.TypeReg},
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
				{Name: "dir/.wh..wh..opq", Type: tar.TypeReg},
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
				{Name: "dir/.wh..wh..opq", Type: tar.TypeReg},
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
				{Name: "dir/.wh..wh..opq", Type: tar.TypeReg},
			},
		},
		"symlink-1": {
			Input: []TestLayer{
				{
					{Name: "foo", Type: tar.TypeSymlink, Linkname: "bar"},
				},
				{
					{Name: "foo", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "bar", Type: tar.TypeReg},
				{Name: "foo", Type: tar.TypeSymlink, Linkname: "bar"},
			},
		},
		"symlink-2": {
			Input: []TestLayer{
				{
					{Name: "foo", Type: tar.TypeSymlink, Linkname: "bar"},
				},
				{
					{Name: "foo", Type: tar.TypeSymlink, Linkname: "baz"},
				},
			},
			Output: TestLayer{
				{Name: "bar", Type: tar.TypeSymlink, Linkname: "baz"},
				{Name: "foo", Type: tar.TypeSymlink, Linkname: "bar"},
			},
		},
		"symlink-3": {
			Input: []TestLayer{
				{
					{Name: "dir", Type: tar.TypeDir},
					{Name: "dir/lnk", Type: tar.TypeSymlink, Linkname: "../tgt"},
				},
				{
					{Name: "dir/lnk", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "dir/", Type: tar.TypeDir},
				{Name: "dir/lnk", Type: tar.TypeSymlink, Linkname: "../tgt"},
				{Name: "tgt", Type: tar.TypeReg},
			},
		},
		"symlink-4": {
			Input: []TestLayer{
				{
					{Name: "dir", Type: tar.TypeDir},
					{Name: "dir/lnk", Type: tar.TypeSymlink, Linkname: "../../tgt"}, // outside of image
				},
				{
					{Name: "dir/lnk", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "dir/", Type: tar.TypeDir},
				{Name: "dir/lnk", Type: tar.TypeReg},
			},
		},
		"symlink-5": {
			Input: []TestLayer{
				{
					{Name: "dir", Type: tar.TypeDir},
					{Name: "dir/lnk", Type: tar.TypeSymlink, Linkname: "/tgt"},
				},
				{
					{Name: "dir/lnk", Type: tar.TypeReg},
				},
			},
			Output: TestLayer{
				{Name: "dir/", Type: tar.TypeDir},
				{Name: "dir/lnk", Type: tar.TypeSymlink, Linkname: "/tgt"},
				{Name: "tgt", Type: tar.TypeReg},
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

			actual, err := Squash(input)
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tc.Output, ParseTestLayer(t, actual))
		})
	}
}
