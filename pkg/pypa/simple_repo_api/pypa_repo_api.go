//Package simple_repo_api implementes the PyPA Simple repository API.
//
// https://packaging.python.org/specifications/simple-repository-api/
package simple_repo_api

import (
	"github.com/datawire/ocibuild/pkg/pep503"
	_ "github.com/datawire/ocibuild/pkg/pep592"
	"github.com/datawire/ocibuild/pkg/pep629"
)

func NewClient() pep503.Client {
	return pep503.Client{
		HTMLHook: pep629.HTMLVersionCheck,
	}
}
