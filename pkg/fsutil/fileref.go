package fsutil

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"sort"
	"time"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

type FileReference interface {
	fs.FileInfo
	FullName() string
	Open() (io.ReadCloser, error)
}

func LayerFromFileReferences(vfs []FileReference, clampTime time.Time, opts ...ociv1tarball.LayerOption) (ociv1.Layer, error) {
	sort.Slice(vfs, func(i, j int) bool {
		return vfs[i].FullName() < vfs[j].FullName()
	})

	var byteWriter bytes.Buffer
	tarWriter := tar.NewWriter(&byteWriter)

	for _, file := range vfs {
		header, err := tar.FileInfoHeader(file, "")
		if err != nil {
			return nil, err
		}
		header.Name = file.FullName()
		if header.ModTime.After(clampTime) {
			header.ModTime = clampTime
		}
		if header.AccessTime.After(clampTime) {
			header.AccessTime = clampTime
		}
		if header.ChangeTime.After(clampTime) {
			header.ChangeTime = clampTime
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeReg {
			fh, err := file.Open()
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(tarWriter, fh); err != nil {
				_ = fh.Close()
				return nil, err
			}
			if err := fh.Close(); err != nil {
				return nil, err
			}
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}

	byteSlice := byteWriter.Bytes()
	return ociv1tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(byteSlice)), nil
	}, opts...)
}
