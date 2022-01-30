// Package pyinspect determines information about a Python environment.
package pyinspect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/datawire/dlib/dexec"

	"github.com/datawire/ocibuild/pkg/python"
	"github.com/datawire/ocibuild/pkg/python/pep425"
)

type FileInfo interface {
	fs.FileInfo
	UID() int
	GID() int
	UName() string
	GName() string
}

type fileInfo struct {
	fs.FileInfo
	uid, gid     int
	uname, gname string
}

func (fi *fileInfo) UID() int      { return fi.uid }
func (fi *fileInfo) GID() int      { return fi.gid }
func (fi *fileInfo) UName() string { return fi.uname }
func (fi *fileInfo) GName() string { return fi.gname }

type FS interface {
	// Split mimics path/filepath.Split.
	Split(path string) (dir, file string)

	// Join mimics path/filepath.Join.
	Join(elem ...string) string

	// Stat mimics os.Stat, but
	//
	//  1. with the additional requirement that name must be an absolute path
	//  2. the FileInfo also exposes ownership information.
	Stat(name string) (FileInfo, error)

	// LookPath mimics os/exec.LookPath, but io/fs.PathError is used instead of exec.Error.
	LookPath(file string) (string, error)
}

// Shebangs takes an interpreter command (like "python3") and turns it in to a pair of paths to put
// after the "#!" in a shebang.
func Shebangs(sys FS, generic string) (console, graphical string, err error) {
	generic, err = sys.LookPath(generic)
	if err != nil {
		return "", "", err
	}

	console = generic
	if dirPart, filePart := sys.Split(console); strings.HasPrefix(filePart, "pythonw") {
		withoutW := sys.Join(dirPart, "python"+strings.TrimPrefix(filePart, "pythonw"))
		if withoutW, err := sys.LookPath(withoutW); err == nil {
			console = withoutW
		}
	}

	graphical = generic
	if dirPart, filePart := sys.Split(console); strings.HasPrefix(filePart, "python") &&
		!strings.HasPrefix(filePart, "pythonw") {
		withW := sys.Join(dirPart, "pythonw"+strings.TrimPrefix(filePart, "python"))
		if withW, err := sys.LookPath(withW); err == nil {
			graphical = withW
		}
	}

	return console, graphical, nil
}

type DynamicInfo struct {
	MagicNumberB64 string
	Tags           pep425.Installer
	VersionInfo    python.VersionInfo
	Scheme         python.Scheme
}

func Dynamic(ctx context.Context, cmdline ...string) (*DynamicInfo, error) {
	cmd := dexec.CommandContext(ctx, cmdline[0], append(cmdline[1:], "-c", `
import json
import sys
from base64 import b64encode
from importlib.util import MAGIC_NUMBER
from packaging.tags import sys_tags
from pip._internal.locations import get_scheme

version_info_slots = ['major', 'minor', 'micro', 'releaselevel', 'serial']

scheme=get_scheme("")

json.dump({
  "MagicNumberB64": b64encode(MAGIC_NUMBER).decode('utf-8'),
  "Tags": [str(tag) for tag in sys_tags()],
  "VersionInfo": {slot: getattr(sys.version_info, slot) for slot in version_info_slots},
  "Scheme": {slot: getattr(scheme, slot) for slot in scheme.__slots__},
}, sys.stdout)
`)...)
	cmd.DisableLogging = true
	bs, err := cmd.Output()
	if err != nil {
		var exitErr *dexec.ExitError
		if errors.As(err, &exitErr) {
			err = fmt.Errorf("%w:\n > %s", err,
				strings.Join(strings.Split(string(exitErr.Stderr), "\n"), "\n > "))
		}
		err = fmt.Errorf("running Python: %w", err)
		return nil, err
	}
	var data DynamicInfo
	if err := json.Unmarshal(bs, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
