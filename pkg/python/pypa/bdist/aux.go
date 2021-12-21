package bdist

import (
	"archive/zip"
	"context"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python"
)

type version []int

var specVersion = version{1, 0}

func parseVersion(str string) (version, error) {
	parts := strings.Split(str, ".")
	ret := make(version, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("could not parse wheel version number: %q: %w", str, err)
		}
		ret = append(ret, n)
	}
	return ret, nil
}

func (v version) String() string {
	parts := make([]string, 0, len(v))
	for _, n := range v {
		parts = append(parts, strconv.Itoa(n))
	}
	return strings.Join(parts, ".")
}

func vercmp(a, b version) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		aPart := 0
		if i < len(a) {
			aPart = a[i]
		}

		bPart := 0
		if i < len(b) {
			bPart = b[i]
		}

		if aPart != bPart {
			return aPart - bPart
		}
	}
	return 0
}

func sanitizePlatformForLayer(plat python.Platform) (python.Platform, error) {
	if err := plat.Init(); err != nil {
		return plat, err
	}
	paths := []*string{
		&plat.Scheme.PureLib,
		&plat.Scheme.PlatLib,
		&plat.Scheme.Headers,
		&plat.Scheme.Scripts,
		&plat.Scheme.Data,
	}
	for _, pathPtr := range paths {
		clean := (*pathPtr)[1:]
		*pathPtr = clean
	}
	return plat, nil
}

// This is based off of pip/_internal/utils/unpacking.py:zip_item_is_executable()`
func isExecutable(f *zip.File) bool { //nolint:deadcode,unused
	externalAttrs := python.ParseZIPExternalAttributes(f.FileHeader.ExternalAttrs)
	return externalAttrs.UNIX.IsRegular() && (externalAttrs.UNIX&0111 != 0)
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

// vfs is a map[filename]FileReference where filename==FileReference.FullName() and filenames use
// forward slashes and are absolute paths but without the leading "/" (same as io/fs).
type PostInstallHook func(ctx context.Context, clampTime time.Time, vfs map[string]fsutil.FileReference, installedDistInfoDir string) error

func PostInstallHooks(hooks ...PostInstallHook) PostInstallHook {
	if len(hooks) == 0 {
		return nil
	}
	return func(ctx context.Context, clampTime time.Time, vfs map[string]fsutil.FileReference, installedDistInfoDir string) error {
		for _, hook := range hooks {
			if err := hook(ctx, clampTime, vfs, installedDistInfoDir); err != nil {
				return err
			}
		}
		return nil
	}
}
