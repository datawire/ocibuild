package fsutil

import (
	"bytes"
	"io"
	"io/fs"
	"os"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

func PathOpener(filename string) ociv1tarball.Opener {
	fi, err := os.Stat(filename)
	if err != nil {
		return func() (io.ReadCloser, error) {
			return nil, err
		}
	}
	if fi.Mode().IsRegular() {
		// Open the file for each access.  This does not work on pipes.
		return func() (io.ReadCloser, error) {
			file, err := os.Open(filename)
			if err != nil {
				return nil, err
			}
			return file, nil
		}
	} else {
		// Read the file in to memory once, and then work on that.  This avoids extra IO,
		// but uses more memory.
		bs, err := os.ReadFile(filename)
		return func() (io.ReadCloser, error) {
			if err != nil {
				return nil, err
			}
			return io.NopCloser(bytes.NewReader(bs)), nil
		}
	}
}

func OpenImage(filename string) (ociv1.Image, error) {
	img, err := ociv1tarball.Image(PathOpener(filename), nil)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "open imagefile",
			Path: filename,
			Err:  err,
		}
	}
	return img, nil
}

func OpenLayer(filename string) (ociv1.Layer, error) {
	layer, err := ociv1tarball.LayerFromOpener(PathOpener(filename))
	if err != nil {
		return nil, &fs.PathError{
			Op:   "open layerfile",
			Path: filename,
			Err:  err,
		}
	}
	return layer, nil
}
