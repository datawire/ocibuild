// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

// Package simple_repo_api implements the PyPA Simple repository API.
//
// https://packaging.python.org/specifications/simple-repository-api/
package simple_repo_api

import (
	"context"
	"fmt"
	"sort"

	"github.com/datawire/ocibuild/pkg/python/pep425"
	"github.com/datawire/ocibuild/pkg/python/pep440"
	"github.com/datawire/ocibuild/pkg/python/pep503"
	"github.com/datawire/ocibuild/pkg/python/pep592"
	"github.com/datawire/ocibuild/pkg/python/pep629"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
)

// The current interface for querying available package versions and retrieving packages from an
// index server is defined in PEP 503, with the addition of “yank” support (allowing a kind of file
// deletion) as defined in PEP 592 and specifying the interface version provided by an index server
// in PEP 629.

type Client struct {
	pep503.Client
	SupportedTags pep425.Installer
}

func NewClient(python *pep440.Version, supportedTags pep425.Installer) Client {
	return Client{
		Client: pep503.Client{
			Python:   python,
			HTMLHook: pep629.HTMLVersionCheck,

			BaseURL:    "",  // default, let user override after initialization
			HTTPClient: nil, // default, let user override after initialization
			UserAgent:  "",  // default, let user override after initialization
		},
		SupportedTags: supportedTags,
	}
}

func (c Client) SelectWheel(ctx context.Context, pkgname string, version pep440.Specifier) (*pep503.FileLink, error) {
	// 0. Filter by pkgname
	links, err := c.ListPackageFiles(ctx, pkgname)
	if err != nil {
		return nil, err
	}
	// 1. Filter by version
	version2links := make(map[string][]pep503.FileLink)
	var whlLinks []pep503.FileLink //nolint:prealloc // 'continue' is quite likely
	var versions []pep440.Version  //nolint:prealloc // 'continue' is quite likely
	for _, link := range links {
		linkInfo, err := bdist.ParseFilename(link.Text)
		if err != nil {
			continue
		}
		if !c.SupportedTags.Supports(linkInfo.CompatibilityTag) {
			continue
		}
		version2links[linkInfo.Version.String()] = append(version2links[linkInfo.Version.String()], link)
		whlLinks = append(whlLinks, link)
		versions = append(versions, linkInfo.Version)
	}
	selectedVersion := version.Select(versions, pep440.MultiExcluder{
		pep440.ExcludePreReleases{
			AllowList: nil, // TODO
		},
		pep592.ExcludeYanked(whlLinks),
	})
	if selectedVersion == nil {
		return nil, fmt.Errorf("no matches for %q %q", pkgname, version.String())
	}
	links = version2links[selectedVersion.String()]
	if len(links) == 1 {
		ret := links[0]
		return &ret, nil
	}
	// 2. Filter by perferred compatibility tag
	var minRank int
	var minList []pep503.FileLink
	for _, link := range links {
		linkInfo, _ := bdist.ParseFilename(link.Text)
		rank := c.SupportedTags.Preference(linkInfo.CompatibilityTag)
		if minRank == 0 || rank < minRank {
			minRank = rank
			minList = nil
		}
		if rank == minRank {
			minList = append(minList, link)
		}
	}
	links = minList
	if len(links) == 1 {
		ret := links[0]
		return &ret, nil
	}
	// 3. Finally, tie-break by build tag.
	sort.Slice(links, func(i, j int) bool {
		iInfo, _ := bdist.ParseFilename(links[i].Text)
		jInfo, _ := bdist.ParseFilename(links[j].Text)
		return iInfo.BuildTag.Cmp(jInfo.BuildTag) < 0
	})
	ret := links[0]
	return &ret, nil
}
