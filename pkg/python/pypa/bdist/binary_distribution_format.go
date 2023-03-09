// This file contains as comments the full text of
// https://github.com/pypa/packaging.python.org/blob/main/source/specifications/binary-distribution-format.rst,
// which has been placed in to the public domain.
//
// It is up-to-date as of 2021-11-13 (commit 526ee6d6563855551bfee4d967a06823044ddbd4, 2021-09-02).

package bdist

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"net/textproto"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/datawire/dlib/derror"
	"github.com/datawire/dlib/dlog"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python"
	"github.com/datawire/ocibuild/pkg/python/pep425"
	"github.com/datawire/ocibuild/pkg/python/pep440"
	"github.com/datawire/ocibuild/pkg/reproducible"
)

//
// .. _binary-distribution-format:
//
// ==========================
// Binary distribution format
// ==========================
//
// The binary distribution format (:term:`wheel <Wheel>`) was originally defined
// in :pep:`427`. The current version of the specification is here.
//
//
// Abstract
// ========
//
// This PEP describes a built-package format for Python called "wheel".
//
// A wheel is a ZIP-format archive with a specially formatted file name and
// the ``.whl`` extension.  It contains a single distribution nearly as it
// would be installed according to PEP 376 with a particular installation
// scheme.  Although a specialized installer is recommended, a wheel file
// may be installed by simply unpacking into site-packages with the standard
// 'unzip' tool while preserving enough information to spread its contents
// out onto their final paths at any later time.
type wheel struct {
	zip *zip.Reader

	cachedDistInfoDir string
}

func (wh *wheel) Open(filename string) (io.ReadCloser, error) {
	filename = path.Clean(filename)
	for _, file := range wh.zip.File {
		if path.Clean(file.Name) == filename {
			return file.Open()
		}
	}
	return nil, fmt.Errorf("%w in wheel zip archive: %q", fs.ErrNotExist, filename)
}

// InstallWheel produces an image layer from a Python wheel file.
//
// If minTime is non-zero, it is used as the timestamp when extracting files from the wheel file; if
// it is zero then the timestamps in the wheel file are preserved.
//
// If maxTime is zero, then it defaults based on the maximum timestamp in the wheel file.
func InstallWheel(
	ctx context.Context,
	plat python.Platform,
	minTime, maxTime time.Time,
	wheelfilename string,
	hook PostInstallHook,
	opts ...ociv1tarball.LayerOption,
) (ociv1.Layer, error) {
	plat, err := sanitizePlatformForLayer(plat)
	if err != nil {
		return nil, fmt.Errorf("bdist.InstallWheel: validate python.Platform: %w", err)
	}

	zipReader, err := zip.OpenReader(wheelfilename)
	if err != nil {
		return nil, fmt.Errorf("bdist.Installwheel: open wheel: %w", err)
	}
	defer zipReader.Close()

	wh := &wheel{ //nolint:varnamelen // same as receiver name
		zip: &zipReader.Reader,

		cachedDistInfoDir: "", // don't know it yet
	}

	if err := wh.integrityCheck(); err != nil {
		return nil, fmt.Errorf("bdist.InstallWheel: wheel integrity: %w", err)
	}

	if maxTime.IsZero() {
		var maxWheelTime time.Time
		for _, file := range wh.zip.File {
			if file.Modified.After(maxWheelTime) {
				maxWheelTime = file.Modified
			}
		}
		if maxWheelTime.IsZero() {
			maxTime = reproducible.Now()
		} else {
			maxWheelTimeRoundedUp := maxWheelTime.Round(time.Second)
			if maxWheelTimeRoundedUp.Before(maxWheelTime) {
				maxWheelTimeRoundedUp.Add(time.Second)
			}
			// Add 1 more second, so that .pyc files have an mtime after their source
			// .py file.
			maxTime = maxWheelTimeRoundedUp.Add(time.Second)
		}
	}

	vfs, installedDistInfoDir, err := wh.installToVFS(ctx, plat, minTime, maxTime)
	if err != nil {
		return nil, fmt.Errorf("bdist.InstallWheel: %w", err)
	}

	if hook != nil {
		if err := hook(ctx, maxTime, vfs, installedDistInfoDir); err != nil {
			return nil, fmt.Errorf("bdist.InstallWheel: post-install hook: %w", err)
		}
	}

	// ensure that parent directories exist
	for filename := range vfs {
		for dir := path.Dir(filename); dir != "."; dir = path.Dir(dir) {
			if _, exists := vfs[dir]; !exists {
				vfs[dir] = &fsutil.InMemFileReference{
					FileInfo: (&tar.Header{
						Typeflag: tar.TypeDir,
						Name:     dir,
						Mode:     0o755,
						ModTime:  maxTime,
					}).FileInfo(),
					MFullName: dir,
					MContent:  nil,
				}
			}
		}
	}

	// chown
	refs := make([]fsutil.FileReference, 0, len(vfs))
	for _, file := range vfs {
		ref, err := newTarEntry(file, func(header *tar.Header) {
			header.Uid = plat.UID
			header.Gid = plat.GID
			header.Uname = plat.UName
			header.Gname = plat.GName
		})
		if err != nil {
			return nil, fmt.Errorf("bdist.InstallWheel: chown: %w", err)
		}
		refs = append(refs, ref)
	}

	layer, err := fsutil.LayerFromFileReferences(refs, maxTime, opts...)
	if err != nil {
		return nil, fmt.Errorf("bdist.InstallWheel: generate layer: %w", err)
	}
	return layer, nil
}

//
//
// PEP Acceptance
// ==============
//
// This PEP was accepted, and the defined wheel version updated to 1.0, by
// Nick Coghlan on 16th February, 2013 [1]_

//nolint:gochecknoglobals // Would be 'const'.
var specVersion, _ = pep440.ParseVersion("1.0")

//
//
// Rationale
// =========
//
// Python needs a package format that is easier to install than sdist.
// Python's sdist packages are defined by and require the distutils and
// setuptools build systems, running arbitrary code to build-and-install,
// and re-compile, code just so it can be installed into a new
// virtualenv.  This system of conflating build-install is slow, hard to
// maintain, and hinders innovation in both build systems and installers.
//
// Wheel attempts to remedy these problems by providing a simpler
// interface between the build system and the installer.  The wheel
// binary package format frees installers from having to know about the
// build system, saves time by amortizing compile time over many
// installations, and removes the need to install a build system in the
// target environment.
//
//
// Details
// =======
//

func (wh *wheel) installToVFS(
	ctx context.Context,
	plat python.Platform,
	minTime,
	maxTime time.Time,
) (map[string]fsutil.FileReference, string, error) {
	// Installing a wheel 'distribution-1.0-py32-none-any.whl'
	// -------------------------------------------------------
	//
	// Wheel installation notionally consists of two phases:
	//
	// - Unpack.
	//
	//   a. Parse ``distribution-1.0.dist-info/WHEEL``.
	metadata, err := wh.parseDistInfoWheel()
	if err != nil {
		return nil, "", fmt.Errorf("parse .dist-info/Wheel: %w", err)
	}
	//   b. Check that installer is compatible with Wheel-Version.  Warn if
	//      minor version is greater, abort if major version is greater.
	wheelVersion, err := pep440.ParseVersion(metadata.Get("Wheel-Version"))
	if err != nil {
		return nil, "", fmt.Errorf("parse Wheel-Version: %w", err)
	}
	if wheelVersion.Major() > specVersion.Major() {
		return nil, "", fmt.Errorf("wheel file's Wheel-Version (%s) is not compatible with this wheel parser",
			wheelVersion)
	}
	if wheelVersion.Cmp(*specVersion) > 0 {
		dlog.Warnf(ctx, "wheel file's Wheel-Version (%s) is newer than this wheel parser", wheelVersion)
	}
	//   c. If Root-Is-Purelib == 'true', unpack archive into purelib
	//      (site-packages).
	//   d. Else unpack archive into platlib (site-packages).
	var dstDir string
	if metadata.Get("Root-Is-Purelib") == "true" {
		dstDir = plat.Scheme.PureLib
	} else {
		dstDir = plat.Scheme.PlatLib
	}
	vfs := make(map[string]fsutil.FileReference)
	for _, file := range wh.zip.File {
		create(vfs, minTime, path.Join(dstDir, file.FileHeader.Name), &zipEntry{
			header: file.FileHeader,
			open:   file.Open,
		})
	}

	//
	// - Spread.
	//
	//   a. Unpacked archive includes ``distribution-1.0.dist-info/`` and (if
	//      there is data) ``distribution-1.0.data/``.
	//   b. Move each subtree of ``distribution-1.0.data/`` onto its
	//      destination path. Each subdirectory of ``distribution-1.0.data/``
	//      is a key into a dict of destination directories, such as
	//      ``distribution-1.0.data/(purelib|platlib|headers|scripts|data)``.
	//      The initially supported paths are taken from
	//      ``distutils.command.install``.
	distInfoDir, err := wh.distInfoDir()
	if err != nil {
		// This already ran successfully inside of .parseDistInfoWheel(); we should get the
		// cached value.
		panic("should not happen")
	}
	vfsTypes := make(map[string]string)
	dataDir := path.Join(dstDir, strings.TrimSuffix(distInfoDir, ".dist-info")+".data")
	for fullName := range vfs {
		if !strings.HasPrefix(fullName, dataDir+"/") {
			continue
		}
		relName := strings.TrimPrefix(fullName, dataDir+"/")
		parts := strings.SplitN(relName, "/", 2)
		key := parts[0]
		var rest string
		if len(parts) > 1 {
			rest = parts[1]
		}

		var dstDataDir string
		switch key {
		case "purelib":
			dstDataDir = plat.Scheme.PureLib
		case "platlib":
			dstDataDir = plat.Scheme.PlatLib
		case "headers":
			dstDataDir = plat.Scheme.Headers
		case "scripts":
			dstDataDir = plat.Scheme.Scripts
		case "data":
			dstDataDir = plat.Scheme.Data
		default:
			return nil, "", fmt.Errorf("unsupported wheel data type %q: %q",
				key, path.Join(strings.TrimSuffix(distInfoDir, ".dist-info")+".data", relName))
		}
		newFullName := path.Join(dstDataDir, rest)
		vfsTypes[newFullName] = key
		if err := rename(vfs, fullName, newFullName); err != nil {
			return nil, "", fmt.Errorf("spread: %w", err)
		}
	}
	//   c. If applicable, update scripts starting with ``#!python`` to point
	//      to the correct interpreter.
	if err := rewritePython(plat, vfs, vfsTypes); err != nil {
		return nil, "", fmt.Errorf("rewrite shebangs: %w", err)
	}
	//   d. Update ``distribution-1.0.dist-info/RECORD`` with the installed
	//      paths.

	delete(vfs, path.Join(dstDir, distInfoDir, "RECORD"))
	delete(vfs, path.Join(dstDir, distInfoDir, "RECORD.jws"))
	delete(vfs, path.Join(dstDir, distInfoDir, "RECORD.p7s"))
	// Intentionally left "TODO"; the spec at
	// https://packaging.python.org/en/latest/specifications/recording-installed-packages/
	// (implemented as a PostInstallHook) overrides us.
	//
	// create(vfs, path.Join(dstDir, distInfoDir, "RECORD"), TODO(vfs))

	//   e. Remove empty ``distribution-1.0.data`` directory.
	delete(vfs, path.Join(dstDir, strings.TrimSuffix(distInfoDir, ".dist-info")+".data"))
	//   f. Compile any installed .py to .pyc. (Uninstallers should be smart
	//      enough to remove .pyc even if it is not mentioned in RECORD.)
	var srcs []fsutil.FileReference //nolint:prealloc // 'continue' is quite likely
	for _, file := range vfs {
		if !strings.HasSuffix(file.Name(), ".py") {
			continue
		}
		srcs = append(srcs, file)
	}
	outs, err := plat.PyCompile(ctx, maxTime, []string{
		plat.Scheme.PureLib,
		plat.Scheme.PlatLib,
	}, srcs)
	if err != nil {
		return nil, "", fmt.Errorf("py_compile: %w", err)
	}
	for _, newFile := range outs {
		vfs[newFile.FullName()] = newFile
	}

	return vfs, path.Join(dstDir, distInfoDir), nil
}

//
// Recommended installer features
// ''''''''''''''''''''''''''''''
//

func rewritePython(plat python.Platform, vfs map[string]fsutil.FileReference, vfsTypes map[string]string) error {
	// Rewrite ``#!python``.
	//     In wheel, scripts are packaged in
	//     ``{distribution}-{version}.data/scripts/``.  If the first line of
	//     a file in ``scripts/`` starts with exactly ``b'#!python'``, rewrite to
	//     point to the correct interpreter.  Unix installers may need to add
	//     the +x bit to these files if the archive was created on Windows.
	//
	//     The ``b'#!pythonw'`` convention is allowed. ``b'#!pythonw'`` indicates
	//     a GUI script instead of a console script.
	for filename, key := range vfsTypes {
		if key != "scripts" {
			continue
		}
		header, err := func() ([]byte, error) {
			fh, err := vfs[filename].Open()
			if err != nil {
				return nil, err
			}
			defer fh.Close()
			return io.ReadAll(io.LimitReader(fh, int64(len("#!pythonw"))))
		}()
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(header, []byte("#!python")) {
			continue
		}

		entry := vfs[filename].(*zipEntry) //nolint:forcetypeassert // it's a bug if it's not true

		originalOpen := entry.open
		shebang := plat.ConsoleShebang
		skip := len("#!python")
		if bytes.Equal(header, []byte("#!pythonw")) {
			skip++
			shebang = plat.GraphicalShebang
		}
		entry.open = func() (io.ReadCloser, error) {
			inner, err := originalOpen()
			if err != nil {
				return nil, err
			}
			return readCloser{
				Reader: io.MultiReader(
					strings.NewReader("#!"+shebang),
					&skipReader{
						skip:  skip,
						inner: inner,
					},
				),
				Closer: inner,
			}, nil
		}
		entry.header.UncompressedSize64 += 2 + uint64(len(shebang))
		entry.header.UncompressedSize64 -= uint64(skip)

		externalAttrs := python.ParseZIPExternalAttributes(entry.header.ExternalAttrs)
		externalAttrs.UNIX |= 0o111
		entry.header.ExternalAttrs = externalAttrs.Raw()
	}
	return nil
}

//
// Generate script wrappers.
//     In wheel, scripts packaged on Unix systems will certainly not have
//     accompanying .exe wrappers.  Windows installers may want to add them
//     during install.
//
// Recommended archiver features
// '''''''''''''''''''''''''''''
//
// Place ``.dist-info`` at the end of the archive.
//     Archivers are encouraged to place the ``.dist-info`` files physically
//     at the end of the archive.  This enables some potentially interesting
//     ZIP tricks including the ability to amend the metadata without
//     rewriting the entire archive.
//
//
// File Format
// -----------
//
// File name convention
// ''''''''''''''''''''
//
// The wheel filename is ``{distribution}-{version}(-{build
// tag})?-{python tag}-{abi tag}-{platform tag}.whl``.
//
// distribution
//     Distribution name, e.g. 'django', 'pyramid'.
//
// version
//     Distribution version, e.g. 1.0.
//
// build tag
//     Optional build number.  Must start with a digit.  Acts as a
//     tie-breaker if two wheel file names are the same in all other
//     respects (i.e. name, version, and other tags).  Sort as an
//     empty tuple if unspecified, else sort as a two-item tuple with
//     the first item being the initial digits as an ``int``, and the
//     second item being the remainder of the tag as a ``str``.
//
// language implementation and version tag
//     E.g. 'py27', 'py2', 'py3'.
//
// abi tag
//     E.g. 'cp33m', 'abi3', 'none'.
//
// platform tag
//     E.g. 'linux_x86_64', 'any'.
//
// For example, ``distribution-1.0-1-py27-none-any.whl`` is the first
// build of a package called 'distribution', and is compatible with
// Python 2.7 (any Python 2.7 implementation), with no ABI (pure Python),
// on any CPU architecture.
//
// The last three components of the filename before the extension are
// called "compatibility tags."  The compatibility tags express the
// package's basic interpreter requirements and are detailed in PEP 425.

type FileNameData struct {
	Distribution     string
	Version          pep440.Version
	BuildTag         *BuildTag
	CompatibilityTag pep425.Tag
}

var reFilename = regexp.MustCompile(regexp.MustCompile(`\s+`).ReplaceAllString(`
		^(?P<distribution>[^-]+)
		-(?P<version>[^-]+)
		(?:-(?P<build_n>[0-9]+)(?P<build_l>[^-0-9][^-]*)?)?
		-(?P<python>[^-]+)
		-(?P<abi>[^-]+)
		-(?P<platform>[^-]+)
		\.whl$`, ``))

func ParseFilename(filename string) (*FileNameData, error) {
	match := reFilename.FindStringSubmatch(filename)
	if match == nil {
		return nil, fmt.Errorf("invalid wheel filename: %q", filename)
	}

	var ret FileNameData

	ret.Distribution = match[reFilename.SubexpIndex("distribution")]

	ver, err := pep440.ParseVersion(match[reFilename.SubexpIndex("version")])
	if err != nil {
		return nil, fmt.Errorf("invalid wheel filename: %q: %w", filename, err)
	}
	ret.Version = *ver

	if buildN := match[reFilename.SubexpIndex("build_n")]; buildN != "" {
		n, _ := strconv.Atoi(buildN)
		ret.BuildTag = &BuildTag{
			Int: n,
			Str: match[reFilename.SubexpIndex("build_l")],
		}
	}

	ret.CompatibilityTag = pep425.Tag{
		Python:   match[reFilename.SubexpIndex("python")],
		ABI:      match[reFilename.SubexpIndex("abi")],
		Platform: match[reFilename.SubexpIndex("platform")],
	}

	return &ret, nil
}

type BuildTag struct {
	Int int
	Str string
}

func (t BuildTag) String() string {
	return fmt.Sprintf("%d%s", t.Int, t.Str)
}

func (a *BuildTag) Cmp(b *BuildTag) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil && b != nil:
		return -1
	case a != nil && b == nil:
		return 1
	}
	if d := a.Int - b.Int; d != 0 {
		return d
	}
	switch {
	case a.Str < b.Str:
		return -1
	case a.Str > b.Str:
		return 1
	default:
		return 0
	}
}

//
// Escaping and Unicode
// ''''''''''''''''''''
//
// As the components of the filename are separated by a dash (``-``, HYPHEN-MINUS),
// this character cannot appear within any component. This is handled as follows:
//

func GenerateFilename(data FileNameData) (string, error) {
	var ret strings.Builder
	// - In distribution names, any run of ``-_.`` characters (HYPHEN-MINUS, LOW LINE
	//   and FULL STOP) should be replaced with ``_`` (LOW LINE). This is equivalent
	//   to :pep:`503` normalisation followed by replacing ``-`` with ``_``.
	ret.WriteString(regexp.MustCompile("[-_.]+").ReplaceAllLiteralString(data.Distribution, "_"))
	// - Version numbers should be normalised according to :pep:`440`. Normalised
	//   version numbers cannot contain ``-``.
	ver, err := data.Version.Normalize()
	if err != nil {
		return "", err
	}
	ret.WriteString("-")
	ret.WriteString(ver.String())
	// - The remaining components may not contain ``-`` characters, so no escaping
	//   is necessary.
	//
	// Tools producing wheels should verify that the filename components do not contain
	// ``-``, as the resulting file may not be processed correctly if they do.
	if data.BuildTag != nil {
		build := data.BuildTag.String()
		if strings.Contains(build, "-") {
			return "", fmt.Errorf("invalid build tag: contains dash: %q", build)
		}
		ret.WriteString("-")
		ret.WriteString(build)
	}
	compat := data.CompatibilityTag.String()
	if strings.Count(compat, "-") != 2 {
		return "", fmt.Errorf("invalid compatibility tag: %q", compat)
	}
	ret.WriteString("-")
	ret.WriteString(compat)
	ret.WriteString(".whl")
	return ret.String(), nil
}

//
// The archive filename is Unicode.  It will be some time before the tools
// are updated to support non-ASCII filenames, but they are supported in
// this specification.
//
// The filenames *inside* the archive are encoded as UTF-8.  Although some
// ZIP clients in common use do not properly display UTF-8 filenames,
// the encoding is supported by both the ZIP specification and Python's
// ``zipfile``.
//
// File contents
// '''''''''''''
//
// The contents of a wheel file, where {distribution} is replaced with the
// name of the package, e.g. ``beaglevote`` and {version} is replaced with
// its version, e.g. ``1.0.0``, consist of:
//
// #. ``/``, the root of the archive, contains all files to be installed in
//    ``purelib`` or ``platlib`` as specified in ``WHEEL``.  ``purelib`` and
//    ``platlib`` are usually both ``site-packages``.
// #. ``{distribution}-{version}.dist-info/`` contains metadata.
// #. ``{distribution}-{version}.data/`` contains one subdirectory
//    for each non-empty install scheme key not already covered, where
//    the subdirectory name is an index into a dictionary of install paths
//    (e.g. ``data``, ``scripts``, ``include``, ``purelib``, ``platlib``).
// #. Python scripts must appear in ``scripts`` and begin with exactly
//    ``b'#!python'`` in order to enjoy script wrapper generation and
//    ``#!python`` rewriting at install time.  They may have any or no
//    extension.
// #. ``{distribution}-{version}.dist-info/METADATA`` is Metadata version 1.1
//    or greater format metadata.

func (wh *wheel) parseDistInfoWheel() (textproto.MIMEHeader, error) {
	// #. ``{distribution}-{version}.dist-info/WHEEL`` is metadata about the archive
	//    itself in the same basic key: value format::
	//
	//        Wheel-Version: 1.0
	//        Generator: bdist_wheel 1.0
	//        Root-Is-Purelib: true
	//        Tag: py2-none-any
	//        Tag: py3-none-any
	//        Build: 1
	infoDir, err := wh.distInfoDir()
	if err != nil {
		return nil, err
	}
	wheelFile, err := wh.Open(path.Join(infoDir, "WHEEL"))
	if err != nil {
		return nil, err
	}
	defer wheelFile.Close()

	// textproto.Reader.ReadMIMEHeader() expects a blank line to mark the end of the header and
	// the start of the body.  But in WHEEL there is no body, so the blank line should be
	// optional.  So use an io.MultiReader to add a few trailing CRLFs to keep ReadMIMEHeader
	// happy no matter what WHEEL's trailing newline situation is.
	kvReader := textproto.NewReader(bufio.NewReader(io.MultiReader(
		wheelFile,
		strings.NewReader("\r\n\r\n\r\n"),
	)))
	return kvReader.ReadMIMEHeader()
	//
	// #. ``Wheel-Version`` is the version number of the Wheel specification.
	// #. ``Generator`` is the name and optionally the version of the software
	//    that produced the archive.
	// #. ``Root-Is-Purelib`` is true if the top level directory of the archive
	//    should be installed into purelib; otherwise the root should be installed
	//    into platlib.
	// #. ``Tag`` is the wheel's expanded compatibility tags; in the example the
	//    filename would contain ``py2.py3-none-any``.
	// #. ``Build`` is the build number and is omitted if there is no build number.
}

// #. A wheel installer should warn if Wheel-Version is greater than the
//    version it supports, and must fail if Wheel-Version has a greater
//    major version than the version it supports.
// #. Wheel, being an installation format that is intended to work across
//    multiple versions of Python, does not generally include .pyc files.
// #. Wheel does not contain setup.py or setup.cfg.
//
// This version of the wheel specification is based on the distutils install
// schemes and does not define how to install files to other locations.
// The layout offers a superset of the functionality provided by the existing
// wininst and egg binary formats.
//
//
// The .dist-info directory
// ^^^^^^^^^^^^^^^^^^^^^^^^
//
// #. Wheel .dist-info directories include at a minimum METADATA, WHEEL,
//    and RECORD.
// #. METADATA is the package metadata, the same format as PKG-INFO as
//    found at the root of sdists.
// #. WHEEL is the wheel metadata specific to a build of the package.
// #. RECORD is a list of (almost) all the files in the wheel and their
//    secure hashes.  Unlike PEP 376, every file except RECORD, which
//    cannot contain a hash of itself, must include its hash.  The hash
//    algorithm must be sha256 or better; specifically, md5 and sha1 are
//    not permitted, as signed wheel files rely on the strong hashes in
//    RECORD to validate the integrity of the archive.

//nolint:gochecknoglobals // Would be 'const'.
var strongHashes = map[string]func() hash.Hash{
	// The spec is an open-ended list of hashes, so here's what PIP 20.3.4
	// pip/_internal/utils/hashes.py includes:
	"sha256": sha256.New,
	"sha384": sha512.New384,
	"sha512": sha512.New,
}

// #. PEP 376's INSTALLER and REQUESTED are not included in the archive.
// #. RECORD.jws is used for digital signatures.  It is not mentioned in
//    RECORD.
// #. RECORD.p7s is allowed as a courtesy to anyone who would prefer to
//    use S/MIME signatures to secure their wheel files.  It is not
//    mentioned in RECORD.
// #. During extraction, wheel installers verify all the hashes in RECORD
//    against the file contents.  Apart from RECORD and its signatures,
//    installation will fail if any file in the archive is not both
//    mentioned and correctly hashed in RECORD.

//
//
// The .data directory
// ^^^^^^^^^^^^^^^^^^^
//
// Any file that is not normally installed inside site-packages goes into
// the .data directory, named as the .dist-info directory but with the
// .data/ extension::
//
//     distribution-1.0.dist-info/
//
//     distribution-1.0.data/
//
// The .data directory contains subdirectories with the scripts, headers,
// documentation and so forth from the distribution.  During installation the
// contents of these subdirectories are moved onto their destination paths.
//
//
// Signed wheel files
// ------------------
//
// Wheel files include an extended RECORD that enables digital
// signatures.  PEP 376's RECORD is altered to include a secure hash
// ``digestname=urlsafe_b64encode_nopad(digest)`` (urlsafe base64
// encoding with no trailing = characters) as the second column instead
// of an md5sum.  All possible entries are hashed, including any
// generated files such as .pyc files, but not RECORD which cannot contain its
// own hash. For example::
//
//     file.py,sha256=AVTFPZpEKzuHr7OvQZmhaU3LvwKz06AJw8mT\_pNh2yI,3144
//     distribution-1.0.dist-info/RECORD,,
//
// The signature file(s) RECORD.jws and RECORD.p7s are not mentioned in
// RECORD at all since they can only be added after RECORD is generated.
// Every other file in the archive must have a correct hash in RECORD
// or the installation will fail.
//
// If JSON web signatures are used, one or more JSON Web Signature JSON
// Serialization (JWS-JS) signatures is stored in a file RECORD.jws adjacent
// to RECORD.  JWS is used to sign RECORD by including the SHA-256 hash of
// RECORD as the signature's JSON payload::
//
//     { "hash": "sha256=ADD-r2urObZHcxBW3Cr-vDCu5RJwT4CaRTHiFmbcIYY" }
//
// (The hash value is the same format used in RECORD.)
//
// If RECORD.p7s is used, it must contain a detached S/MIME format signature
// of RECORD.
//
// A wheel installer is not required to understand digital signatures but
// MUST verify the hashes in RECORD against the extracted file contents.
// When the installer checks file hashes against RECORD, a separate signature
// checker only needs to establish that RECORD matches the signature.
//
// See
//
// - https://self-issued.info/docs/draft-ietf-jose-json-web-signature.html
// - https://self-issued.info/docs/draft-jones-jose-jws-json-serialization.html
// - https://self-issued.info/docs/draft-ietf-jose-json-web-key.html
// - https://self-issued.info/docs/draft-jones-jose-json-private-key.html
//
//

func (wh *wheel) integrityCheck() error {
	distInfoDir, err := wh.distInfoDir()
	if err != nil {
		return err
	}

	todo := make(map[string]struct{})
	for _, file := range wh.zip.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := path.Clean(file.Name)
		switch name {
		case path.Join(distInfoDir, "RECORD.jws"):
			// skip
		case path.Join(distInfoDir, "RECORD.p7s"):
			// skip
		default:
			todo[name] = struct{}{}
		}
	}

	recordData, err := func() ([][]string, error) {
		recordName := path.Join(distInfoDir, "RECORD")
		reader, err := wh.Open(recordName)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = reader.Close()
		}()
		data, err := csv.NewReader(reader).ReadAll()
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", recordName, err)
		}
		return data, nil
	}()
	if err != nil {
		return err
	}

	checkFile := func(filename, algo string) (hashsum string, size int64, err error) {
		reader, err := wh.Open(filename)
		if err != nil {
			return "", 0, err
		}
		defer func() {
			_ = reader.Close()
		}()

		var (
			hasher hash.Hash
			dst    = io.Discard
		)
		if algo != "" {
			newHasher, ok := strongHashes[algo]
			if !ok {
				return "", 0, fmt.Errorf("unsupported hash algorithm: %q", algo)
			}
			hasher = newHasher()
			dst = hasher
		}

		size, err = io.Copy(dst, reader)
		if err != nil {
			return "", 0, err
		}

		if hasher != nil {
			hashsum = algo + "=" + base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
		}

		return hashsum, size, err
	}

	var errs derror.MultiError
	for i, row := range recordData {
		if len(row) != 3 {
			errs = append(errs, fmt.Errorf("RECORD row %d: does not have 3 columns: %q", i, row))
			continue
		}
		name, recHashsum, recSize := path.Clean(row[0]), row[1], row[2]
		delete(todo, name)
		if recHashsum == "" || recSize == "" {
			switch name {
			case path.Join(distInfoDir, "RECORD"):
				// skip
			default:
				errs = append(errs, fmt.Errorf("RECORD row %d: missing hash or size: %q", i, row))
			}
		}

		algo := strings.SplitN(recHashsum, "=", 2)[0]
		actHashsum, actSize, err := checkFile(name, algo)
		if err != nil {
			errs = append(errs, fmt.Errorf("RECORD row %d: file %q: %w",
				i, name, err))
			continue
		}
		if recHashsum != "" && actHashsum != recHashsum {
			errs = append(errs, fmt.Errorf("RECORD row %d: file %q: checksum mismatch: RECORD=%q actual=%q",
				i, name, recHashsum, actHashsum))
		}
		if recSize != "" && strconv.FormatInt(actSize, 10) != recSize {
			errs = append(errs, fmt.Errorf("RECORD row %d: file %q: size mismatch: RECORD=%s actual=%d",
				i, name, recSize, actSize))
		}
	}

	if len(todo) > 0 {
		todoNames := make([]string, 0, len(todo))
		for name := range todo {
			todoNames = append(todoNames, name)
		}
		sort.Strings(todoNames)
		errs = append(errs, fmt.Errorf("files not mentioned in RECORD: %q", todoNames))
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// Comparison to .egg
// ------------------
//
// #. Wheel is an installation format; egg is importable.  Wheel archives
//    do not need to include .pyc and are less tied to a specific Python
//    version or implementation. Wheel can install (pure Python) packages
//    built with previous versions of Python so you don't always have to
//    wait for the packager to catch up.
// #. Wheel uses .dist-info directories; egg uses .egg-info.  Wheel is
//    compatible with the new world of Python packaging and the new
//    concepts it brings.
// #. Wheel has a richer file naming convention for today's
//    multi-implementation world.  A single wheel archive can indicate
//    its compatibility with a number of Python language versions and
//    implementations, ABIs, and system architectures.  Historically the
//    ABI has been specific to a CPython release, wheel is ready for the
//    stable ABI.
// #. Wheel is lossless.  The first wheel implementation bdist_wheel
//    always generates egg-info, and then converts it to a .whl.  It is
//    also possible to convert existing eggs and bdist_wininst
//    distributions.
// #. Wheel is versioned.  Every wheel file contains the version of the
//    wheel specification and the implementation that packaged it.
//    Hopefully the next migration can simply be to Wheel 2.0.
// #. Wheel is a reference to the other Python.
//
//
// FAQ
// ===
//
//
// Wheel defines a .data directory.  Should I put all my data there?
// -----------------------------------------------------------------
//
//     This specification does not have an opinion on how you should organize
//     your code.  The .data directory is just a place for any files that are
//     not normally installed inside ``site-packages`` or on the PYTHONPATH.
//     In other words, you may continue to use ``pkgutil.get_data(package,
//     resource)`` even though *those* files will usually not be distributed
//     in *wheel's* ``.data`` directory.
//
//
// Why does wheel include attached signatures?
// -------------------------------------------
//
//     Attached signatures are more convenient than detached signatures
//     because they travel with the archive.  Since only the individual files
//     are signed, the archive can be recompressed without invalidating
//     the signature or individual files can be verified without having
//     to download the whole archive.
//
//
// Why does wheel allow JWS signatures?
// ------------------------------------
//
//     The JOSE specifications of which JWS is a part are designed to be easy
//     to implement, a feature that is also one of wheel's primary design
//     goals.  JWS yields a useful, concise pure-Python implementation.
//
//
// Why does wheel also allow S/MIME signatures?
// --------------------------------------------
//
//     S/MIME signatures are allowed for users who need or want to use
//     existing public key infrastructure with wheel.
//
//     Signed packages are only a basic building block in a secure package
//     update system.  Wheel only provides the building block.
//
//
// What's the deal with "purelib" vs. "platlib"?
// ---------------------------------------------
//
//     Wheel preserves the "purelib" vs. "platlib" distinction, which is
//     significant on some platforms. For example, Fedora installs pure
//     Python packages to '/usr/lib/pythonX.Y/site-packages' and platform
//     dependent packages to '/usr/lib64/pythonX.Y/site-packages'.
//
//     A wheel with "Root-Is-Purelib: false" with all its files
//     in ``{name}-{version}.data/purelib`` is equivalent to a wheel with
//     "Root-Is-Purelib: true" with those same files in the root, and it
//     is legal to have files in both the "purelib" and "platlib" categories.
//
//     In practice a wheel should have only one of "purelib" or "platlib"
//     depending on whether it is pure Python or not and those files should
//     be at the root with the appropriate setting given for "Root-is-purelib".
//
//
// Is it possible to import Python code directly from a wheel file?
// ----------------------------------------------------------------
//
//     Technically, due to the combination of supporting installation via
//     simple extraction and using an archive format that is compatible with
//     ``zipimport``, a subset of wheel files *do* support being placed directly
//     on ``sys.path``. However, while this behaviour is a natural consequence
//     of the format design, actually relying on it is generally discouraged.
//
//     Firstly, wheel *is* designed primarily as a distribution format, so
//     skipping the installation step also means deliberately avoiding any
//     reliance on features that assume full installation (such as being able
//     to use standard tools like ``pip`` and ``virtualenv`` to capture and
//     manage dependencies in a way that can be properly tracked for auditing
//     and security update purposes, or integrating fully with the standard
//     build machinery for C extensions by publishing header files in the
//     appropriate place).
//
//     Secondly, while some Python software is written to support running
//     directly from a zip archive, it is still common for code to be written
//     assuming it has been fully installed. When that assumption is broken
//     by trying to run the software from a zip archive, the failures can often
//     be obscure and hard to diagnose (especially when they occur in third
//     party libraries). The two most common sources of problems with this
//     are the fact that importing C extensions from a zip archive is *not*
//     supported by CPython (since doing so is not supported directly by the
//     dynamic loading machinery on any platform) and that when running from
//     a zip archive the ``__file__`` attribute no longer refers to an
//     ordinary filesystem path, but to a combination path that includes
//     both the location of the zip archive on the filesystem and the
//     relative path to the module inside the archive. Even when software
//     correctly uses the abstract resource APIs internally, interfacing with
//     external components may still require the availability of an actual
//     on-disk file.
//
//     Like metaclasses, monkeypatching and metapath importers, if you're not
//     already sure you need to take advantage of this feature, you almost
//     certainly don't need it. If you *do* decide to use it anyway, be
//     aware that many projects will require a failure to be reproduced with
//     a fully installed package before accepting it as a genuine bug.
//
// Changes
// =======
//
// Since :pep:`427`, this specification has changed as follows:
//
// - The rules on escaping in wheel filenames were revised, to bring them into line
//   with what popular tools actually do (February 2021).
//
//
// References
// ==========
//
// .. [1] PEP acceptance
//    (https://mail.python.org/pipermail/python-dev/2013-February/124103.html)
//
//
// Appendix
// ========
//
// Example urlsafe-base64-nopad implementation::
//
//     # urlsafe-base64-nopad for Python 3
//     import base64
//
//     def urlsafe_b64encode_nopad(data):
//         return base64.urlsafe_b64encode(data).rstrip(b'=')
//
//     def urlsafe_b64decode_nopad(data):
//         pad = b'=' * (4 - (len(data) & 3))
//         return base64.urlsafe_b64decode(data + pad)
//
//
// Copyright
// =========
//
// This document has been placed into the public domain.
