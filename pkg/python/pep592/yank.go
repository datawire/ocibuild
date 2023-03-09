// Copyright (C) 2021  Ambassador Labs
//
// SPDX-License-Identifier: Apache-2.0

// Package pep592 implements PEP 592 -- Adding "Yank" Support to the Simple API.
//
// https://www.python.org/dev/peps/pep-0592/
package pep592

import (
	"github.com/datawire/ocibuild/pkg/python/pep440"
	"github.com/datawire/ocibuild/pkg/python/pep503"
	"github.com/datawire/ocibuild/pkg/python/pypa/bdist"
)

func IsYanked(l pep503.FileLink) bool {
	_, yanked := l.DataAttrs["data-yanked"]
	return yanked
}

type excludeYanked struct {
	yankedVersions map[string]struct{}
}

func ExcludeYanked(links []pep503.FileLink) pep440.ExclusionBehavior {
	ret := excludeYanked{
		yankedVersions: make(map[string]struct{}),
	}
	for _, link := range links {
		if IsYanked(link) {
			fileInfo, err := bdist.ParseFilename(link.Text)
			if err != nil {
				continue
			}
			ret.yankedVersions[fileInfo.Version.String()] = struct{}{}
		}
	}
	return ret
}

func (e excludeYanked) Allow(v pep440.Version) bool {
	_, yanked := e.yankedVersions[v.String()]
	return yanked
}
