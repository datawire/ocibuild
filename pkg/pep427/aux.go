package pep427

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/datawire/layertool/pkg/python"
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
	Target struct {
		// For shebangs
		Python string // /usr/lib/python3

		// Installation directories: These are the directories described in
		// distutils.command.install.SCHEME_KEYS and
		// distutils.command.install.INSTALL_SCHEMES.
		PureLib string // /usr/lib/python3.9/site-packages
		PlatLib string // /usr/lib64/python3.9/site-packages
		Headers string // /usr/include/python3.9/$name/ (e.g. $name=cpython)
		Scripts string // /usr/bin
		Data    string // /usr
	}

	// For byte-compiling
	Python string // /usr/lib/python3
}

// This is based off of pip/_internal/utils/unpacking.py:zip_item_is_executable()`
func isExecutable(f *zip.File) bool {
	externalAttrs := python.ParseZIPExternalAttributes(f.FileHeader.ExternalAttrs)
	return externalAttrs.UNIX.IsRegular() && (externalAttrs.UNIX&0111 != 0)
}

// distInfoDir returns the "{name}.dist-info" directory for the wheel file.
//
// This is based off of `pip/_internal/utils/wheel.py:wheel_dist_info_dir()`, since PEP 427 doesn't
// actually have much to say about resolving ambiguity.
func (wh *wheel) distInfoDir() (string, error) {
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

type zipEntry struct {
	zip.FileHeader
	Open func() (io.ReadCloser, error)
}

func TODO() *zipEntry {
	return nil
}
