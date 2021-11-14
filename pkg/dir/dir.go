// Package dir deals with creating a layer from a directory
package dir

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

func LayerFromDir(dirname string, opts ...ociv1tarball.LayerOption) (ociv1.Layer, error) {
	type logEntry struct {
		Name string
		Info fs.FileInfo
	}

	var byteWriter bytes.Buffer
	tarWriter := tar.NewWriter(&byteWriter)

	var log []logEntry

	err := filepath.Walk(dirname, func(p string, info fs.FileInfo, e error) error {
		if e != nil {
			return e
		}
		name, err := filepath.Rel(dirname, p)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		defer func() {
			log = append(log, logEntry{
				Name: name,
				Info: info,
			})
		}()

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = name
		for _, entry := range log {
			if os.SameFile(entry.Info, info) {
				header.Typeflag = tar.TypeLink
				header.Linkname = entry.Name
				break
			}
		}
		if header.Typeflag == tar.TypeLink {
			header.Linkname, err = os.Readlink(p)
			if err != nil {
				return err
			}
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if header.Typeflag == tar.TypeReg {
			fh, err := os.Open(p)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tarWriter, fh); err != nil {
				_ = fh.Close()
				return err
			}
			if err := fh.Close(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}

	byteSlice := byteWriter.Bytes()
	return ociv1tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(byteSlice)), nil
	}, opts...)
}
