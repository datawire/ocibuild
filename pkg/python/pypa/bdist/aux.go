package bdist

import (
	"archive/zip"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

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

type Platform struct {
	ConsoleShebang   string // "/usr/bin/python3"
	GraphicalShebang string // "/usr/bin/python3"

	Scheme Scheme

	UID   int
	GID   int
	UName string
	GName string

	PyCompile python.Compiler `json:"-"`
}

type Scheme struct {
	// Installation directories: These are the directories described in
	// distutils.command.install.SCHEME_KEYS and
	// distutils.command.install.INSTALL_SCHEMES.
	PureLib string `json:"purelib"` // "/usr/lib/python3.9/site-packages"
	PlatLib string `json:"platlib"` // "/usr/lib64/python3.9/site-packages"
	Headers string `json:"headers"` // "/usr/include/python3.9/$name/" (e.g. $name=cpython)
	Scripts string `json:"scripts"` // "/usr/bin"
	Data    string `json:"data"`    // "/usr"
}

func sanitizePlatformForLayer(plat Platform) (Platform, error) {
	if plat.ConsoleShebang == "" && plat.GraphicalShebang == "" {
		return plat, fmt.Errorf("Platform specification does not specify a path to use for shebangs")
	}
	if plat.ConsoleShebang == "" {
		plat.ConsoleShebang = plat.GraphicalShebang
	}
	if plat.GraphicalShebang == "" {
		plat.GraphicalShebang = plat.ConsoleShebang
	}
	for _, pair := range []struct {
		name string
		ptr  *string
	}{
		{"purelib", &plat.Scheme.PureLib},
		{"platlib", &plat.Scheme.PlatLib},
		{"headers", &plat.Scheme.Headers},
		{"scripts", &plat.Scheme.Scripts},
		{"data", &plat.Scheme.Data},
	} {
		if !path.IsAbs(*pair.ptr) {
			return plat, fmt.Errorf("Platform install scheme %q is not an absolute path: %q", pair.name, *pair.ptr)
		}
		clean := (*pair.ptr)[1:]
		*pair.ptr = clean
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
