// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

// Package dir deals with creating a layer from a directory.
package dir

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

type Prefix struct {
	DirName string

	Mode int

	Ownership
}

type Ownership struct {
	UID   int
	UName string

	GID   int
	GName string
}

func LayerFromDir(
	dirname string,
	prefix *Prefix,
	chown *Ownership,
	clampTime time.Time,
	opts ...ociv1tarball.LayerOption,
) (ociv1.Layer, error) {
	type logEntry struct {
		Name string
		Info fs.FileInfo
	}

	var byteWriter bytes.Buffer
	tarWriter := tar.NewWriter(&byteWriter)

	var log []logEntry

	if prefix != nil {
		if prefix.Mode == 0 {
			prefix.Mode = 0o755
		}
		var dirs []string
		for dir := prefix.DirName; dir != "."; dir = path.Dir(dir) {
			dirs = append(dirs, dir)
		}
		for i := len(dirs) - 1; i >= 0; i-- {
			if err := tarWriter.WriteHeader(&tar.Header{
				Name:     dirs[i],
				Typeflag: tar.TypeDir,
				ModTime:  clampTime,

				Mode:  int64(prefix.Mode),
				Uid:   prefix.UID,
				Uname: prefix.UName,
				Gid:   prefix.GID,
				Gname: prefix.GName,
			}); err != nil {
				return nil, err
			}
		}
	}

	err := filepath.Walk(dirname, func(filename string, info fs.FileInfo, e error) error {
		if e != nil {
			return e
		}
		name, err := filepath.Rel(dirname, filename)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		if name == "." {
			return nil
		}
		if prefix != nil {
			name = path.Join(prefix.DirName, name)
		}
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
		if header.Typeflag == tar.TypeSymlink {
			header.Linkname, err = os.Readlink(filename)
			if err != nil {
				return err
			}
		}
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
			return err
		}
		if chown != nil {
			if chown.UID >= 0 {
				header.Uid = chown.UID
			}
			if chown.UName != "" {
				header.Uname = chown.UName
			}
			if chown.GID >= 0 {
				header.Gid = chown.GID
			}
			if chown.GName != "" {
				header.Gname = chown.GName
			}
		}
		if header.Typeflag == tar.TypeReg {
			reader, err := os.Open(filename)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tarWriter, reader); err != nil {
				_ = reader.Close()
				return err
			}
			if err := reader.Close(); err != nil {
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
