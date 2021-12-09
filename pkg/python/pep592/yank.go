// Package pep592 implements Python PEP 592 -- Adding "Yank" Support to the Simple API.
//
// https://www.python.org/dev/peps/pep-0592/
package pep592

import (
	"github.com/datawire/ocibuild/pkg/python/pep503"
)

func IsYanked(l pep503.FileLink) bool {
	_, yanked := l.DataAttrs["data-yanked"]
	return yanked
}
