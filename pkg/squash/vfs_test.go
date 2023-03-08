// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package squash_test

import (
	"archive/tar"
	"errors"
	"path"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/squash"
)

func TestVFS(t *testing.T) {
	t.Parallel()
	testcases := map[string]TestLayer{
		"1": {
			{Name: ".", Type: tar.TypeDir},
			{Name: "foo", Type: tar.TypeDir},
			{Name: "foo/.aaa", Type: tar.TypeReg},
			{Name: "foo/bar", Type: tar.TypeReg},
			{Name: "foo/baz", Type: tar.TypeReg},
			{Name: "foo/d", Type: tar.TypeReg},
			{Name: "foo/moved", Type: tar.TypeReg},
			{Name: "foo/zap", Type: tar.TypeReg},
			{Name: "sym", Type: tar.TypeSymlink, Linkname: "foo"},
		},
	}
	for tcName, tc := range testcases {
		tc := tc
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()
			filenames := make([]string, 0, len(tc)-1)
			for _, file := range tc {
				name := path.Clean(file.Name)
				if name == "." {
					continue
				}
				filenames = append(filenames, name)
			}
			layer := tc.ToLayer(t)

			vfs, err := squash.Load([]ociv1.Layer{layer}, false)
			require.NoError(t, err)

			err = fstest.TestFS(vfs, filenames...)
			// Filter out some errors https://github.com/golang/go/issues/50401
			if err != nil {
				ignoreRE := regexp.MustCompile(`^(.+): Open\+ReadAll: read (.+): is a directory$`)
				var lines []string
				for _, line := range strings.Split(err.Error(), "\n") {
					ignore := false
					if m := ignoreRE.FindStringSubmatch(line); m != nil {
						for _, tfile := range tc {
							if path.Clean(tfile.Name) == path.Clean(m[2]) {
								ignore = true
							}
						}
					}
					if !ignore {
						lines = append(lines, line)
					}
				}
				if len(lines) <= 1 {
					err = nil
				} else {
					err = errors.New(strings.Join(lines, "\n"))
				}
			}
			require.NoError(t, err)
		})
	}
}
