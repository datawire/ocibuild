package main

import (
	"bytes"
	"io"
	"io/fs"
	"os"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func PathOpener(filename string) tarball.Opener {
	bs, err := os.ReadFile(filename)
	return func() (io.ReadCloser, error) {
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(bs)), nil
	}
}

func OpenImage(filename string) (v1.Image, error) {
	img, err := tarball.Image(PathOpener(filename), nil)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "open imagefile",
			Path: filename,
			Err:  err,
		}
	}
	return img, nil
}

func OpenLayer(filename string) (v1.Layer, error) {
	layer, err := tarball.LayerFromOpener(PathOpener(filename))
	if err != nil {
		return nil, &fs.PathError{
			Op:   "open layerfile",
			Path: filename,
			Err:  err,
		}
	}
	return layer, nil
}
