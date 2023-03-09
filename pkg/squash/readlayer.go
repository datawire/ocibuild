// Copyright (C) 2020-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package squash

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
)

type fileEntry struct {
	Header *tar.Header
	Body   []byte
}

type layerFS struct {
	WhiteoutMarkers []fileEntry
	Files           []fileEntry
}

// parseLayer parses a Layer in to a filesystem object, with the following sanitizations made for
// consistent querying:
//
//  - Paths are always path.Clean()'d (notably, directories do NOT contain trailing "/").
func parseLayer(layer ociv1.Layer, omitContent bool) (*layerFS, error) {
	lfs := new(layerFS)
	layerReader, err := layer.Uncompressed()
	if err != nil {
		return nil, fmt.Errorf("reading layer contents: %w", err)
	}
	defer layerReader.Close()
	tarReader := tar.NewReader(layerReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("reading tar: %w", err)
		}

		cleanName := path.Clean(header.Name)
		if strings.HasPrefix(cleanName, "/") || strings.HasPrefix(cleanName, "../") || cleanName == ".." {
			return nil, fmt.Errorf("layer contains file outside of image root: %q", header.Name)
		}
		header.Name = cleanName

		var body []byte
		if omitContent {
			// #nosec G110 -- mitigated with io.Discard
			if _, err := io.Copy(io.Discard, tarReader); err != nil {
				return nil, fmt.Errorf("reading tar: %w", err)
			}
		} else {
			body, err = io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("reading tar: %w", err)
			}
		}
		entry := fileEntry{
			Header: header,
			Body:   body,
		}
		if strings.HasPrefix(path.Base(header.Name), ".wh.") {
			lfs.WhiteoutMarkers = append(lfs.WhiteoutMarkers, entry)
		} else {
			lfs.Files = append(lfs.Files, entry)
		}
	}
	return lfs, nil
}
