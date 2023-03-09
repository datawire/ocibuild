// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package bdist

import (
	"archive/zip"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python"
)

func sanitizePlatformForLayer(plat python.Platform) (python.Platform, error) {
	if err := plat.Init(); err != nil {
		return plat, err
	}
	// transform the paths from `path/filepath` paths to `io/fs` paths.
	paths := []*string{
		&plat.Scheme.PureLib,
		&plat.Scheme.PlatLib,
		&plat.Scheme.Headers,
		&plat.Scheme.Scripts,
		&plat.Scheme.Data,
	}
	for _, pathPtr := range paths {
		clean := strings.TrimPrefix(filepath.ToSlash(*pathPtr), "/")
		*pathPtr = clean
	}

	return plat, nil
}

// This is based off of pip/_internal/utils/unpacking.py:zip_item_is_executable()`
func isExecutable(fh zip.FileHeader) bool {
	externalAttrs := python.ParseZIPExternalAttributes(fh.ExternalAttrs)
	return externalAttrs.UNIX.IsRegular() && (externalAttrs.UNIX&0o111 != 0)
}

// distInfoDir returns the "{name}.dist-info" directory for the wheel file.
//
// This is based off of `pip/_internal/utils/wheel.py:wheel_dist_info_dir()`, since PEP 427 doesn't
// actually have much to say about resolving ambiguity.
func (wh *wheel) distInfoDir() (string, error) {
	if wh.cachedDistInfoDir != "" {
		return wh.cachedDistInfoDir, nil
	}
	infoDirs := make(map[string]struct{})
	for _, file := range wh.zip.File {
		dirname := strings.Split(path.Clean(file.FileHeader.Name), "/")[0]
		if !strings.HasSuffix(dirname, ".dist-info") {
			continue
		}
		infoDirs[dirname] = struct{}{}
	}

	switch len(infoDirs) {
	case 0:
		return "", fmt.Errorf(".dist-info directory not found")
	case 1:
		for infoDir := range infoDirs {
			wh.cachedDistInfoDir = infoDir
			return infoDir, nil
		}
		panic("not reached")
	default:
		list := make([]string, 0, len(infoDirs))
		for dir := range infoDirs {
			list = append(list, dir)
		}
		sort.Strings(list)
		return "", fmt.Errorf("multiple .dist-info directories found: %v", list)
	}
}

// vfs is a map[filename]FileReference where filename==FileReference.FullName().
//
// As a reminder, FileFeference.FullName() returns io/fs paths: (1) forward-slashes and (2) absolute
// paths but without the leading "/".
type PostInstallHook func(
	ctx context.Context,
	clampTime time.Time,
	vfs map[string]fsutil.FileReference,
	installedDistInfoDir string,
) error

func PostInstallHooks(hooks ...PostInstallHook) PostInstallHook {
	if len(hooks) == 0 {
		return nil
	}
	return func(
		ctx context.Context,
		clampTime time.Time,
		vfs map[string]fsutil.FileReference,
		installedDistInfoDir string,
	) error {
		for _, hook := range hooks {
			if err := hook(ctx, clampTime, vfs, installedDistInfoDir); err != nil {
				return err
			}
		}
		return nil
	}
}
