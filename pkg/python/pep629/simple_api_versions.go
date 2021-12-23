// Package pep629 implements PEP 629 -- Versioning PyPI's Simple API.
//
// https://www.python.org/dev/peps/pep-0629/
package pep629

import (
	"context"
	"fmt"

	"github.com/datawire/dlib/dlog"
	"golang.org/x/net/html"

	"github.com/datawire/ocibuild/pkg/htmlutil"
	"github.com/datawire/ocibuild/pkg/python/pep440"
)

//nolint:gochecknoglobals // Would be 'const'.
var SupportedVersion, _ = pep440.ParseVersion("1.0")

func GetVersion(doc *html.Node) (*pep440.Version, error) {
	// <meta name="pypi:repository-version" content="1.0">
	var verStr string
	err := htmlutil.VisitHTML(doc, nil, func(node *html.Node) error {
		if node.Type != html.ElementNode || node.Data != "meta" {
			return nil
		}
		name, _ := htmlutil.GetAttr(node, "", "name")
		if name != "pypi:repository-version" {
			return nil
		}
		_verStr, ok := htmlutil.GetAttr(node, "", "content")
		if ok {
			verStr = _verStr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if verStr == "" {
		verStr = "1.0"
	}
	return pep440.ParseVersion(verStr)
}

func HTMLVersionCheck(ctx context.Context, doc *html.Node) error {
	version, err := GetVersion(doc)
	if err != nil {
		return err
	}
	if version.Major() > SupportedVersion.Major() {
		return fmt.Errorf("server's pypi:repository version (%s) is not compatible with this client", version)
	}
	if version.Minor() > SupportedVersion.Minor() {
		dlog.Warnf(ctx, "server's pypi:repository version (%s) is newer than this client", version)
	}
	return nil
}
