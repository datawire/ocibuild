// Package pep427 implements Python PEP 427 -- The Wheel Binary Package Format 1.0.
//
// https://www.python.org/dev/peps/pep-0427/
package pep427

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/textproto"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/datawire/layertool/pkg/pep425"
	"github.com/datawire/layertool/pkg/python"
)

type wheel struct {
	zip *zip.Reader
}

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

		// Installation directories
		PureLib string // /usr/lib/python3.9/site-packages
		PlatLib string // /usr/lib64/python3.9/site-packages
		Headers string // /usr/include/python3.9/$name/
		Scripts string // /usr/bin
		Data    string // /usr
	}

	// For byte-compiling
	Python sring // /usr/lib/python3

	Mkdir func(string) error
}

func InstallWheel(ctx context.Context, wheelfilename string) error {
	zipReader, err := zip.OpenReader(wheelfilename)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	wh := &wheel{
		zip: &zipReader.Reader,
	}

	// Installing a wheel 'distribution-1.0-py32-none-any.whl'
	// -------------------------------------------------------
	//
	// Wheel installation notionally consists of two phases:
	//
	// - Unpack.
	//   1. Parse `distribution-1.0.dist-info/WHEEL`.
	metadata, err := wh.parseDistInfoWheel()
	if err != nil {
		return err
	}
	//   2. Check that installer is compatible with Wheel-Version. Warn if minor version is
	//      greater, abort if major version is greater.
	wheelVersion, err := parseVersion(metadata.Get("Wheel-Version"))
	if err != nil {
		return err
	}
	if wheelVersion[0] > specVersion[0] {
		return fmt.Errorf("wheel file's Wheel-Version (%s) is not compatible with this wheel parser", wheelVersion)
	}
	if vercmp(wheelVersion, specVersion) > 0 {
		dlog.Warnf(ctx, "wheel file's Wheel-Version (%s) is newer than this wheel parser", wheelVersion)
	}
	if metadata.Get("Root-Is-Purelib") == "true" {
		//   3. If Root-Is-Purelib == 'true', unpack archive into purelib (site-packages).
	} else {
		//   4. Else unpack archive into platlib (site-packages).
	}
	// - Spread.
	//   1. Unpacked archive includes `distribution-1.0.dist-info/` and (if there is data)
	//      `distribution-1.0.data/`.
	//   2. Move each subtree of `distribution-1.0.data/` onto its destination path. Each
	//      subdirectory of `distribution-1.0.data/` is a key into a dict of destination
	//      directories, such as
	//      `distribution-1.0.data/(purelib|platlib|headers|scripts|data)`. The initially
	//      supported paths are taken from `distutils.command.install`.
	//   3. If applicable, update scripts starting with `#!python` to point to the correct
	//      interpreter.
	//   4. Update `distribution-1.0.dist-info/RECORD` with the installed paths.
	//   5. Remove empty `distribution-1.0.data` directory.
	//   6. Compile any installed .py to .pyc. (Uninstallers should be smart enough to remove
	//      .pyc even if it is not mentioned in RECORD.)
	return nil
}

// This is based off of pip/_internal/utils/unpacking.py:zip_item_is_executable()`
func isExecutable(f *zip.File) bool {
	externalAttrs := python.ParseZIPExternalAttributes(f.FileHeader.ExternalAttrs)
	return externalAttrs.UNIX.IsRegular() && (externalAttrs.UNIX&0111 != 0)
}

// distInfoDir returns the "{name}.info-dir" directory for the wheel file.
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

func (wh *wheel) Open(filename string) (io.ReadCloser, error) {
	filename = path.Clean(filename)
	for _, file := range wh.zip.File {
		if path.Clean(file.Name) == filename {
			return file.Open()
		}
	}
	return nil, fmt.Errorf("file does not exist in wheel zip archive: %q", filename)
}

func (wh *wheel) parseDistInfoWheel() (textproto.MIMEHeader, error) {
	infoDir, err := wh.distInfoDir()
	if err != nil {
		return nil, err
	}
	wheelFile, err := wh.Open(path.Join(infoDir, "WHEEL"))
	if err != nil {
		return nil, err
	}
	defer wheelFile.Close()

	kvReader := textproto.NewReader(bufio.NewReader(wheelFile))
	return kvReader.ReadMIMEHeader()
}
