// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

package pyinspect

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/datawire/dlib/dexec"
)

type NativeFS struct{}

var _ FS = NativeFS{}

func (NativeFS) Split(path string) (dir, file string) { return filepath.Split(path) }
func (NativeFS) Join(elem ...string) string           { return filepath.Join(elem...) }
func (NativeFS) Stat(name string) (FileInfo, error) {
	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	fileinfo, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	raw := fileinfo.Sys().(*syscall.Stat_t) //nolint:forcetypeassert // if not, this is a bug and it should crash
	usr, err := user.LookupId(fmt.Sprintf("%v", raw.Uid))
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}
	grp, err := user.LookupGroupId(fmt.Sprintf("%v", raw.Gid))
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}
	return &fileInfo{
		FileInfo: fileinfo,
		uid:      int(raw.Uid),
		gid:      int(raw.Gid),
		uname:    usr.Username,
		gname:    grp.Name,
	}, nil
}

func (NativeFS) LookPath(file string) (string, error) {
	val, err := dexec.LookPath(file)
	if err != nil {
		//nolint:errorlint // We don't want to discard wrappers (except dexec.Error itself).
		if eerr, ok := err.(*dexec.Error); ok {
			err = &fs.PathError{
				Op:   "lookpath",
				Path: file,
				Err:  eerr.Err,
			}
		}
	}
	return val, err
}
