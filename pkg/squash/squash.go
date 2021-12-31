// Package squash deals with squashing multiple layers together in to a single layer.
//
// https://github.com/opencontainers/image-spec/blob/main/layer.md
package squash

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

func loadLayers(layers []ociv1.Layer, omitContent bool) (*fsfile, error) {
	root := &fsfile{ //nolint:exhaustivestruct
		name: ".",
	}
	root.parent = root
	// Apply all the layers
	for _, layer := range layers {
		layerFS, err := parseLayer(layer, omitContent)
		if err != nil {
			return nil, err
		}
		for _, wh := range layerFS.WhiteoutMarkers {
			vfsFile, err := fsGet(root, wh.Header.Name, true, false)
			if err != nil {
				return nil, err
			}
			if err := vfsFile.Set(wh.Header, wh.Body); err != nil {
				return nil, err
			}
		}
		for _, file := range layerFS.Files {
			vfsFile, err := fsGet(root, file.Header.Name, true, false)
			if err != nil {
				return nil, err
			}
			if err := vfsFile.Set(file.Header, file.Body); err != nil {
				return nil, err
			}
		}
	}
	return root, nil
}

// Squash multiple layers together in to a single layer.
//
// This is very similar to github.com/google/go-containerregistry/pkg/v1/mutate.Extract, however:
//
//  1. Includes whiteout markers in the output, since we don't assume to have the root layer.
//  2. Squash properly implements "opaque whiteouts", which go-containerregistry doesn't support.
func Squash(layers []ociv1.Layer, opts ...ociv1tarball.LayerOption) (ociv1.Layer, error) {
	// Load the layers.
	root, err := loadLayers(layers, false)
	if err != nil {
		return nil, err
	}

	// Generate the layer tarball
	var byteWriter bytes.Buffer
	tarWriter := tar.NewWriter(&byteWriter)
	if err := root.WriteTo(tarWriter); err != nil {
		return nil, err
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}

	// Wrap that in to a Layer object
	byteSlice := byteWriter.Bytes()
	return ociv1tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(byteSlice)), nil
	}, opts...)
}

// Load multiple layers as a filesystem.
func Load(layers []ociv1.Layer, omitContent bool) (fs.FS, error) {
	root, err := loadLayers(layers, omitContent)
	if err != nil {
		return nil, err
	}
	return root, nil
}
