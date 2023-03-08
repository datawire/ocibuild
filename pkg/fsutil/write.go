// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package fsutil

import (
	"io"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
)

func WriteLayer(layer ociv1.Layer, dst io.Writer) (err error) {
	layerReader, err := layer.Uncompressed()
	if err != nil {
		return err
	}
	defer func() {
		if _err := layerReader.Close(); _err != nil && err == nil {
			err = _err
		}
	}()
	if _, err := io.Copy(dst, layerReader); err != nil {
		return err
	}
	return nil
}
