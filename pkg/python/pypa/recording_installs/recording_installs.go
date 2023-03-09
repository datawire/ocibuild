// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

// Package simple_repo_api implements the PyPA specification Recording installed projects.
//
// https://packaging.python.org/en/latest/specifications/recording-installed-packages/
package recording_installs

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"hash"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
	"github.com/datawire/ocibuild/pkg/python/pypa/direct_url"
)

const defaultHashAlgorithm = "sha256"

func recordFile(file fsutil.FileReference, hashName string, hasher hash.Hash, baseDir string) ([]string, error) {
	fpName, err := filepath.Rel(filepath.FromSlash("/"+baseDir), filepath.FromSlash("/"+file.FullName()))
	if err != nil {
		return nil, err
	}
	name := filepath.ToSlash(fpName)
	var hash, size string
	if !strings.HasSuffix(name, ".pyc") {
		hasher.Reset()
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = reader.Close()
		}()
		if _, err := io.Copy(hasher, reader); err != nil {
			return nil, err
		}
		hash = hashName + "=" + base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
		size = strconv.FormatInt(file.Size(), 10)
	}
	return []string{name, hash, size}, nil
}

func Record(hashName, installer string, urlData *direct_url.DirectURL) bdist.PostInstallHook {
	return func(
		ctx context.Context,
		clampTime time.Time,
		vfs map[string]fsutil.FileReference,
		installedDistInfoDir string,
	) error {
		// 1. The .dist-info directory

		// Trust the wheel to have the correct .dist-info dir.

		// 2. The METADATA file

		// Trust the wheel to have METADATA.

		// 4. The INSTALLER file
		content := []byte(installer + "\n")
		header := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     path.Join(installedDistInfoDir, "INSTALLER"),
			Mode:     0o644,
			Size:     int64(len(content)),
			ModTime:  clampTime,
		}
		vfs[header.Name] = &fsutil.InMemFileReference{
			FileInfo:  header.FileInfo(),
			MFullName: header.Name,
			MContent:  content,
		}

		// 5. The direct_url.json file
		if urlData != nil {
			if err := direct_url.Record(*urlData)(ctx, clampTime, vfs, installedDistInfoDir); err != nil {
				return fmt.Errorf("recording-installed-packages: direct_url.json: %w", err)
			}
		}

		// 3. The RECORD file
		// Do this last.
		if hashName == "" {
			hashName = defaultHashAlgorithm
		}
		newHasher, ok := python.HashlibAlgorithmsGuaranteed[hashName]
		if !ok {
			return fmt.Errorf("recording-installed-packages: unsupported hash algorithm: %q", hashName)
		}
		hasher := newHasher()
		csvData := [][]string{
			{path.Join(path.Base(installedDistInfoDir), "RECORD"), "", ""},
		}
		for _, file := range vfs {
			if file.IsDir() {
				continue
			}
			row, err := recordFile(file, hashName, hasher, path.Dir(installedDistInfoDir))
			if err != nil {
				return fmt.Errorf("recording installed-packaged: recording file %q: %w",
					file.FullName(), err)
			}
			csvData = append(csvData, row)
		}
		sort.Slice(csvData, func(i, j int) bool {
			return csvData[i][0] < csvData[j][0]
		})
		var recordBytes bytes.Buffer
		csvWriter := csv.NewWriter(&recordBytes)
		csvWriter.UseCRLF = true
		if err := csvWriter.WriteAll(csvData); err != nil {
			return err
		}
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return err
		}
		header = &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     path.Join(installedDistInfoDir, "RECORD"),
			Mode:     0o644,
			Size:     int64(recordBytes.Len()),
			ModTime:  clampTime,
		}
		vfs[header.Name] = &fsutil.InMemFileReference{
			FileInfo:  header.FileInfo(),
			MFullName: header.Name,
			MContent:  recordBytes.Bytes(),
		}

		return nil
	}
}
