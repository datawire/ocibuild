// Copyright (C) 2021-2022  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

// vfs_fs.go is an adapter for the virtual filesystem in vfs.go that adapts it to implement the
// (read-only) stdlib io/fs.FS interface.

package squash

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
	"sync"
	"syscall"
)

// These are the interfaces that we're implementing.
var (
	_ fs.FS          = (*fsfile)(nil)       // from vfs.go; essentially an inode
	_ fs.File        = (*fsfileReader)(nil) // an open file handle
	_ fs.ReadDirFile = (*fsfileReader)(nil)
)

var (
	ErrIsDir   = syscall.EISDIR
	ErrMissing = errors.New("not in layers")
)

// Open implements io/fs.FS.
func (f *fsfile) Open(name string) (_ fs.File, err error) {
	defer func() {
		if err != nil {
			err = &fs.PathError{
				Op:   "open",
				Path: name,
				Err:  err,
			}
		}
	}()
	if !fs.ValidPath(name) {
		return nil, fs.ErrInvalid
	}
	lnk, err := fsGet(f, name, false, false)
	if err != nil {
		return nil, err
	}
	tgt, err := lnk.Get(".", false, true)
	if err != nil {
		return nil, err
	}
	return &fsfileReader{ //nolint:exhaustivestruct
		origName: name,
		tgt:      tgt,
		lnk:      lnk,
	}, nil
}

type fsfileReader struct {
	// static info
	origName string  // filename for use in error messages
	tgt      *fsfile // file info (follow symlinks)
	lnk      *fsfile // file info (don't follow symlinks)

	// dynamic state
	mu      sync.Mutex
	pos     int
	closed  bool
	dirents []fs.DirEntry // cache; generated from .tgt.children
}

// Stat implements io/fs.File.
func (f *fsfileReader) Stat() (fs.FileInfo, error) {
	if f.tgt.header == nil {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: f.origName,
			Err:  ErrMissing,
		}
	}
	hdr := *f.tgt.header // shallow copy
	hdr.Name = f.lnk.header.Name
	return hdr.FileInfo(), nil
}

// Read implements io/fs.File.
func (f *fsfileReader) Read(buf []byte) (_ int, err error) {
	defer func() {
		if err != nil && !errors.Is(err, io.EOF) {
			err = &fs.PathError{
				Op:   "read",
				Path: f.origName,
				Err:  err,
			}
		}
	}()
	if f.tgt.header == nil || f.tgt.header.Typeflag == tar.TypeDir {
		return 0, ErrIsDir
	}
	if int64(len(f.tgt.body)) < f.tgt.header.Size {
		return 0, ErrMissing
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return 0, fs.ErrClosed
	}
	if f.pos == len(f.tgt.body) {
		return 0, io.EOF
	}
	n := copy(buf, f.tgt.body[f.pos:])
	f.pos += n
	return n, nil
}

// Close implements io/fs.File.
func (f *fsfileReader) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return fs.ErrClosed
	}
	f.closed = true
	return nil
}

// missingDirEntry is an fs.DirEntry implementation for directories that are missing their
// `fsfile.header`.
type missingDirEntry string

func (name missingDirEntry) Name() string          { return string(name) }
func (missingDirEntry) IsDir() bool                { return true }
func (missingDirEntry) Type() fs.FileMode          { return fs.ModeDir }
func (missingDirEntry) Info() (fs.FileInfo, error) { return nil, ErrMissing }

func (name missingDirEntry) String() string { return fmt.Sprintf("<missingDirEntry %q>", string(name)) }

// dirEntry is an fs.DirEntry implementation for direcories that have their `fsfile.header`.
type dirEntry struct {
	tgt fs.DirEntry // file info (follow symlinks)
	lnk fs.DirEntry // file info (don't follow symlinks)

	// If you're thinking "hey, wait a minute, `.tgt` is never used, why do we need it at all?",
	// the answer is that because the correct behavior of io/fs.FS isn't clear about how to
	// behave with symlinks there are other reasonable interpretations that would require
	// `.tgt`, and so I'm leaving it in until we get a clear answer.  Notably, if I wanted to
	// get the failing testing/fstest.TestFS tests (see vfs_test.go) to pass, we'd need to make
	// use of `.tgt`, but fixing those would cause other TestFS tests to fail; no way to get
	// them all to pass righ now if there are symlinks to directories.
	//
	// https://github.com/golang/go/issues/45470
	// https://github.com/golang/go/issues/50401
}

func (d *dirEntry) Name() string               { return d.lnk.Name() }
func (d *dirEntry) IsDir() bool                { return d.lnk.IsDir() }
func (d *dirEntry) Type() fs.FileMode          { return d.lnk.Type() }
func (d *dirEntry) Info() (fs.FileInfo, error) { return d.lnk.Info() }

func (d *dirEntry) String() string { return fmt.Sprintf("squash.dirEntry{%q}", d.Name()) }

// ReadDir implements io/fs.ReadDirFile.
func (f *fsfileReader) ReadDir(n int) ([]fs.DirEntry, error) { //nolint:varnamelen // obvious
	if f.tgt.header != nil && f.tgt.header.Typeflag != tar.TypeDir {
		return nil, ErrNotDir
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil, fs.ErrClosed
	}
	if f.dirents == nil {
		f.dirents = make([]fs.DirEntry, 0, len(f.tgt.children))
		for childname, lnk := range f.tgt.children {
			if strings.HasPrefix(childname, ".wh.") {
				continue
			}

			var lnkDirEnt fs.DirEntry
			if lnk.header != nil {
				lnkDirEnt = fs.FileInfoToDirEntry(lnk.header.FileInfo())
			} else {
				lnkDirEnt = missingDirEntry(childname)
			}

			tgt, err := lnk.Get(".", false, true) // follow symlinks
			if err != nil {
				return nil, err
			}
			var tgtDirEnt fs.DirEntry
			if tgt.header != nil {
				tgtDirEnt = fs.FileInfoToDirEntry(tgt.header.FileInfo())
			} else {
				tgtDirEnt = missingDirEntry(childname)
			}

			f.dirents = append(f.dirents, &dirEntry{
				lnk: lnkDirEnt,
				tgt: tgtDirEnt,
			})
		}
		sort.Slice(f.dirents, func(i, j int) bool {
			return f.dirents[i].Name() < f.dirents[j].Name()
		})
	}
	if f.pos > len(f.dirents) {
		f.pos = len(f.dirents)
	}
	ret := f.dirents[f.pos:]
	if n > 0 {
		if n < len(ret) {
			ret = ret[:n]
		}
		if len(ret) == 0 {
			return nil, io.EOF
		}
	}
	if len(ret) == 0 {
		ret = nil
	}
	f.pos += len(ret)
	return ret, nil
}

func (f *fsfileReader) String() string {
	return fmt.Sprintf("<fsfileReader %p %q : pos=%d>", f, f.origName, f.pos)
}
