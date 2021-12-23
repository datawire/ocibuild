package python

import (
	"fmt"
	"path/filepath"
)

type Platform struct {
	ConsoleShebang   string // "/usr/bin/python3"
	GraphicalShebang string // "/usr/bin/python3"

	Scheme Scheme

	UID   int
	GID   int
	UName string
	GName string

	PyCompile Compiler `json:"-"`
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
