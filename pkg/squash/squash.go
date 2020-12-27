package squash

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"sort"
	"strings"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

type fileEntry struct {
	Header *tar.Header
	Body   []byte
}

type layerFS struct {
	WhiteoutMarkers []fileEntry
	Files           []fileEntry
}

func cleanPath(filename string) string {
	ret := path.Clean(filename)
	switch {
	case strings.HasPrefix(ret, "/"):
		return ""
	case strings.HasPrefix(ret, "../"):
		return ""
	case ret != ".":
		ret = "./" + ret
	}
	return ret
}

func sortFilenames(filenames []string) {
	sort.Slice(filenames, func(a, b int) bool {
		return filenameLess(filenames[a], filenames[b])
	})
}

func filenameLess(a, b string) bool {
	aParts := strings.Split(cleanPath(a), "/")
	bParts := strings.Split(cleanPath(b), "/")
	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		// Rule 1: "foo/bar" is before "foo/bar/baz"
		switch {
		case i >= len(aParts):
			return true
		case i >= len(bParts):
			return false
		}
		// Rule 2: Among siblings, whiteout markers are before regular files.
		aWhiteout := strings.HasPrefix(aParts[i], ".wh.")
		bWhiteout := strings.HasPrefix(bParts[i], ".wh.")
		switch {
		case aWhiteout && !bWhiteout:
			return true
		case !aWhiteout && bWhiteout:
			return false
		}
		// Rule 3: Simple string compare
		switch {
		case aParts[i] < bParts[i]:
			return true
		case aParts[i] > bParts[i]:
			return false
		}
		// Rule 4: continue to check the next segment
	}
	return false
}

// parseLayer parses a Layer in to a filesystem object, with the following sanitizations made for
// consistent querying:
//
//  - Paths always start with "." or "./".
//  - Other than that, paths are cleaned and contain no "." or ".." segments.
//  - Directories do not contain trailing "/".
func parseLayer(layer ociv1.Layer) (*layerFS, error) {
	fs := &layerFS{}
	layerReader, err := layer.Uncompressed()
	if err != nil {
		return nil, fmt.Errorf("reading layer contents: %w", err)
	}
	defer layerReader.Close()
	tarReader := tar.NewReader(layerReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("reading tar: %w", err)
		}

		cleanName := cleanPath(header.Name)
		if cleanName == "" {
			return nil, fmt.Errorf("layer contains file outside of image root: %q", header.Name)
		}
		header.Name = cleanName

		body, err := ioutil.ReadAll(tarReader)
		if err != nil {
			return nil, fmt.Errorf("reading tar: %w", err)
		}
		entry := fileEntry{
			Header: header,
			Body:   body,
		}
		if strings.HasPrefix(path.Base(header.Name), ".wh.") {
			fs.WhiteoutMarkers = append(fs.WhiteoutMarkers, entry)
		} else {
			fs.Files = append(fs.Files, entry)
		}
	}
	return fs, nil
}

// Squash multiple layers together in to a single layer.
//
// This is very similar to github.com/google/go-containerregistry/pkg/v1/mutate.Extract, however:
//
//  1. Includes whiteout markers in the output, since we don't assume to have the root layer.
//  2. Squash properly implmenets "opaque whiteouts", which go-containerregistry doesn't support.
func Squash(layers []ociv1.Layer, opts ...ociv1tarball.LayerOption) (ociv1.Layer, error) {
	fs := make(map[string]fileEntry)
	// Apply all the layers
	for _, layer := range layers {
		layerFS, err := parseLayer(layer)
		if err != nil {
			return nil, err
		}
		for _, wh := range layerFS.WhiteoutMarkers {
			dir, base := path.Split(wh.Header.Name)
			if base == ".wh..wh..opq" {
				for k := range fs {
					if strings.HasPrefix(k, dir+"/") {
						delete(fs, k)
					}
				}
			} else {
				delete(fs, dir+strings.TrimPrefix(base, ".wh."))
			}
			fs[wh.Header.Name] = wh
		}
		for _, file := range layerFS.Files {
			if file.Header.Typeflag != tar.TypeDir {
				// Changing a directory to a non-directory will implicitly whiteout
				// anything in that directory.
				for k := range fs {
					if strings.HasPrefix(k, file.Header.Name+"/") {
						delete(fs, k)
					}
				}
			} else if old, ok := fs[file.Header.Name]; ok && old.Header.Typeflag != tar.TypeDir {
				// If we previously changed a directory to a non-directory, then
				// changing it back risks discarding that implicit whiteout, so if
				// we change a file to a directory, make a potential previous
				// implicit whiteout explicit.  I say "potential" because without
				// the entire layer stack (which this function explicitly doesn't
				// require), we can't know if any given file was such a
				// dir-to-non-dir conversion.
				fs[file.Header.Name+"/.wh..wh..opq"] = fileEntry{
					Header: &tar.Header{
						Typeflag: tar.TypeReg,
						Name:     file.Header.Name + "/.wh..wh..opq",
						Size:     0,
						Mode:     0644,
					},
				}
			}

			// Store the file
			fs[file.Header.Name] = file

			// Delete any now-obsolete whiteout files
			dir, base := path.Split(file.Header.Name)
			delete(fs, dir+".wh."+base)

			// A file named "foo/bar" implies that "foo" is a directory, so implicitly
			// white out "foo" if it isn't a directory.
			for {
				slash := strings.LastIndex(dir, "/")
				if slash < 0 {
					break
				}
				dir = dir[:slash]
				dirEnt, ok := fs[dir]
				if ok && dirEnt.Header.Typeflag != tar.TypeDir {
					delete(fs, dir)
					fs[dir+"/.wh..wh..opq"] = fileEntry{
						Header: &tar.Header{
							Typeflag: tar.TypeReg,
							Name:     dir + "/.wh..wh..opq",
							Size:     0,
							Mode:     0644,
						},
					}
				}
			}
		}
	}

	// Sort the filenames per best practices.
	filenames := make([]string, 0, len(fs))
	for fname := range fs {
		filenames = append(filenames, fname)
	}
	sortFilenames(filenames)

	// Generate the layer tarball
	var byteWriter bytes.Buffer
	tarWriter := tar.NewWriter(&byteWriter)
	for _, fname := range filenames {
		file := fs[fname]

		file.Header.Name = strings.TrimPrefix(file.Header.Name, "./")
		if file.Header.Typeflag == tar.TypeDir {
			file.Header.Name += "/"
		}

		if err := tarWriter.WriteHeader(file.Header); err != nil {
			return nil, err
		}

		if _, err := tarWriter.Write(file.Body); err != nil {
			return nil, err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}

	// Wrap that in to a Layer object
	byteSlice := byteWriter.Bytes()
	return ociv1tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader(byteSlice)), nil
	}, opts...)
}
