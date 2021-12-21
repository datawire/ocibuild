// Package direct_url implementes the PyPA specification Recording the Direct URL Origin of
// installed distributions (AKA PEP 610).
//
// https://packaging.python.org/en/latest/specifications/direct-url/
package direct_url

import (
	"archive/tar"
	"context"
	"path"
	"time"

	"github.com/datawire/ocibuild/pkg/fsutil"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
)

type DirectURL struct {
	URL         string       `json:"url"`
	VCSInfo     *VCSInfo     `json:"vcs_info,omitempty"`     // if URL is a VCS reference
	ArchiveInfo *ArchiveInfo `json:"archive_info,omitempty"` // if URL is a sdist or bdist
	DirInfo     *DirInfo     `json:"dir_info,omitempty"`     // if URL is a local directory
}

type VCSInfo struct {
	VCS               string `json:"vcs"`
	RequestedRevision string `json:"requested_revision,omitempty"`
	CommitID          string `json:"commit_id"`
}

type ArchiveInfo struct {
	Hash string `json:"hash,omitempty"`
}

type DirInfo struct {
	Editable bool `json:"editable,omitempty"`
}

func Record(urlData DirectURL) bdist.PostInstallHook {
	return func(ctx context.Context, clampTime time.Time, vfs map[string]fsutil.FileReference, installedDistInfoDir string) error {
		bs, err := jsonDumps(urlData)
		if err != nil {
			return err
		}
		header := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     path.Join(installedDistInfoDir, "direct_url.json"),
			Mode:     0644,
			Size:     int64(len(bs)),
			ModTime:  clampTime,
		}
		vfs[header.Name] = &fsutil.InMemFileReference{
			FileInfo:  header.FileInfo(),
			MFullName: header.Name,
			MContent:  bs,
		}
		return nil
	}
}
