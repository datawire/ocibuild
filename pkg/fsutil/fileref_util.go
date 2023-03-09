// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package fsutil

import (
	"bytes"
	"io"
	"io/fs"
	"path"
)

type InMemFileReference struct {
	fs.FileInfo
	MFullName string
	MContent  []byte
}

func (fr *InMemFileReference) FullName() string { return fr.MFullName }
func (fr *InMemFileReference) Name() string     { return path.Base(fr.MFullName) }
func (fr *InMemFileReference) Open() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(fr.MContent)), nil
}

var _ FileReference = (*InMemFileReference)(nil)
