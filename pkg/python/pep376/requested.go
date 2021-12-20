// Package pep376 implements the REQUESTED metadata of PEP 375 -- Database of Installed Python
// Distributions.
//
// https://packaging.python.org/en/latest/specifications/recording-installed-packages/
package pep376

import (
	"archive/tar"
	"context"
	"path"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
)

func RecordRequested(requested string) bdist.PostInstallHook {
	return func(ctx context.Context, vfs map[string]fsutil.FileReference, installedDistInfoDir string) error {
		// REQUESTED
		// ---------
		//
		// Some install tools automatically detect unfulfilled dependencies and
		// install them. In these cases, it is useful to track which
		// distributions were installed purely as a dependency, so if their
		// dependent distribution is later uninstalled, the user can be alerted
		// of the orphaned dependency.
		//
		// If a distribution is installed by direct user request (the usual
		// case), a file REQUESTED is added to the .dist-info directory of the
		// installed distribution. The REQUESTED file may be empty, or may
		// contain a marker comment line beginning with the "#" character.
		//
		// If an install tool installs a distribution automatically, as a
		// dependency of another distribution, the REQUESTED file should not be
		// created.
		//
		// The ``install`` command of distutils by default creates the REQUESTED
		// file. It accepts ``--requested`` and ``--no-requested`` options to explicitly
		// specify whether the file is created.
		//
		// If a distribution that was already installed on the system as a dependency
		// is later installed by name, the distutils ``install`` command will
		// create the REQUESTED file in the .dist-info directory of the existing
		// installation.
		content := []byte{}
		if requested != "" {
			content = []byte(requested + "\n")
		}
		fullname := path.Join(installedDistInfoDir, "REQUESTED")
		header := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     fullname,
			Mode:     0644,
			Size:     int64(len(content)),
		}
		vfs[fullname] = &fsutil.InMemFileReference{
			FileInfo:  header.FileInfo(),
			MFullName: fullname,
			MContent:  content,
		}

		return nil
	}
}
