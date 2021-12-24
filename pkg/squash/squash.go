// Package squash deals with squashing multiple layers together in to a single layer.
package squash

import (
	"archive/tar"
	"bytes"
	"io"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

// Squash multiple layers together in to a single layer.
//
// This is very similar to github.com/google/go-containerregistry/pkg/v1/mutate.Extract, however:
//
//  1. Includes whiteout markers in the output, since we don't assume to have the root layer.
//  2. Squash properly implements "opaque whiteouts", which go-containerregistry doesn't support.
func Squash(layers []ociv1.Layer, opts ...ociv1tarball.LayerOption) (ociv1.Layer, error) {
	root := &fsfile{ //nolint:exhaustivestruct
		name: ".",
	}
	// Apply all the layers
	for _, layer := range layers {
		layerFS, err := parseLayer(layer)
		if err != nil {
			return nil, err
		}
		for _, wh := range layerFS.WhiteoutMarkers {
			fsGet(root, wh.Header.Name).Set(wh.Header, wh.Body)
		}
		for _, file := range layerFS.Files {
			fsGet(root, file.Header.Name).Set(file.Header, file.Body)
		}
	}

	// Generate the layer tarball
	var byteWriter bytes.Buffer
	tarWriter := tar.NewWriter(&byteWriter)
	if err := root.WriteTo(".", tarWriter); err != nil {
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
