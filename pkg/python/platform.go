package python

import (
	"fmt"
	"path/filepath"

	"github.com/datawire/ocibuild/pkg/python/pep425"
	"github.com/datawire/ocibuild/pkg/python/pep440"
)

type Platform struct {
	ConsoleShebang   string // "/usr/bin/python3"
	GraphicalShebang string // "/usr/bin/python3"

	Scheme Scheme

	UID   int
	GID   int
	UName string
	GName string

	VersionInfo *VersionInfo
	MagicNumber []byte
	Tags        pep425.Installer

	PyCompile Compiler `json:"-" yaml:"-"`
}

type VersionInfo struct {
	Major        int    `json:"major"`
	Minor        int    `json:"minor"`
	Micro        int    `json:"micro"`
	ReleaseLevel string `json:"releaselevel"` // 'alpha', 'beta', 'candidate', or 'final'
	Serial       int    `json:"serial"`
}

func (vi VersionInfo) PEP440() (*pep440.Version, error) {
	var ret pep440.Version
	ret.Release = []int{
		vi.Major,
		vi.Minor,
		vi.Micro,
	}
	switch vi.ReleaseLevel {
	case "alpha":
		ret.Pre = &pep440.PreRelease{L: "a", N: 0}
	case "beta":
		ret.Pre = &pep440.PreRelease{L: "b", N: 0}
	case "candidate":
		ret.Pre = &pep440.PreRelease{L: "rc", N: 0}
	case "final":
		ret.Pre = nil
	default:
		return nil, fmt.Errorf("python.VersionInfo.PEP440: invalid version_info.releaselevel: %q",
			vi.ReleaseLevel)
	}
	return &ret, nil
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

// Init normalizes the shebangs and validates that the scheme has absolute paths.
func (plat *Platform) Init() error {
	if plat.ConsoleShebang == "" && plat.GraphicalShebang == "" {
		return fmt.Errorf("Platform specification does not specify a path to use for shebangs")
	}
	if plat.ConsoleShebang == "" {
		plat.ConsoleShebang = plat.GraphicalShebang
	}
	if plat.GraphicalShebang == "" {
		plat.GraphicalShebang = plat.ConsoleShebang
	}
	for _, pair := range []struct {
		name string
		val  string
	}{
		{"purelib", plat.Scheme.PureLib},
		{"platlib", plat.Scheme.PlatLib},
		{"headers", plat.Scheme.Headers},
		{"scripts", plat.Scheme.Scripts},
		{"data", plat.Scheme.Data},
	} {
		if !filepath.IsAbs(pair.val) {
			return fmt.Errorf("Platform install scheme %q is not an absolute path: %q", pair.name, pair.val)
		}
	}
	return nil
}
