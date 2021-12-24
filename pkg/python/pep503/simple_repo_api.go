// Package pep503 implements PEP 503 -- Simple Repository API.
//
// https://www.python.org/dev/peps/pep-0503/
package pep503

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/datawire/ocibuild/pkg/python/pep345"
	"github.com/datawire/ocibuild/pkg/python/pep440"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
	Python     *pep440.Version
	HTMLHook   func(context.Context, *html.Node) error
}

const PyPIBaseURL = "https://pypi.org/simple/"

func (c *Client) fillDefaults() {
	if c.BaseURL == "" {
		c.BaseURL = PyPIBaseURL
	}
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if c.UserAgent == "" {
		c.UserAgent = "github.com/datawire/ocibuild/pkg/python/pep503"
	}
}

type HTTPError struct {
	Status     string
	StatusCode int
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %s", e.Status)
}

func (c Client) get(ctx context.Context, requestURL string) (_ *url.URL, _ []byte, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("GET %q => %w", requestURL, err)
		}
	}()
	c.fillDefaults()

	// 1. Build the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	// 2. Do the networking
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, nil, err
	}
	if err := resp.Body.Close(); err != nil {
		return nil, nil, err
	}

	// 3. Validate the result
	if resp.StatusCode != http.StatusOK {
		return nil, nil, &HTTPError{Status: resp.Status, StatusCode: resp.StatusCode}
	}
	if u, err := url.Parse(requestURL); err == nil && u.Fragment != "" {
		if keyvals, err := url.ParseQuery(u.Fragment); err == nil {
			for key, vals := range keyvals {
				var sum []byte
				for _, val := range vals {
					switch key {
					case "md5":
						_sum := md5.Sum(content)
						sum = _sum[:]
					case "sha1":
						_sum := sha1.Sum(content)
						sum = _sum[:]
					case "sha224":
						_sum := sha256.Sum224(content)
						sum = _sum[:]
					case "sha256":
						_sum := sha256.Sum256(content)
						sum = _sum[:]
					case "sha384":
						_sum := sha512.Sum384(content)
						sum = _sum[:]
					case "sha512":
						_sum := sha512.Sum512(content)
						sum = _sum[:]
					}
					if sum != nil && hex.EncodeToString(sum) != val {
						//nolint:lll // error string
						return nil, nil, fmt.Errorf("checksum mismatch: %s: expected=%s actual=%s",
							key, val, hex.EncodeToString(sum))
					}
				}
			}
		}
	}

	return resp.Request.URL, content, nil
}

func visitHTML(node *html.Node, before, after func(*html.Node) error) error {
	if before != nil {
		if err := before(node); err != nil {
			return err
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if err := visitHTML(child, before, after); err != nil {
			return err
		}
	}
	if after != nil {
		if err := after(node); err != nil {
			return err
		}
	}
	return nil
}

type Link struct {
	Text      string
	HRef      string
	DataAttrs map[string]string
}

func (c Client) getHTML5Index(ctx context.Context, requestURL string) ([]Link, error) {
	location, content, err := c.get(ctx, requestURL)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	if c.HTMLHook != nil {
		if err := c.HTMLHook(ctx, doc); err != nil {
			return nil, err
		}
	}

	var links []Link
	if err := visitHTML(doc, nil, func(node *html.Node) error {
		if node.Type != html.ElementNode || node.Data != "a" {
			return nil
		}
		link := Link{
			DataAttrs: make(map[string]string),
		}
		for _, attr := range node.Attr {
			switch {
			case attr.Namespace == "" && attr.Key == "href":
				href, err := location.Parse(attr.Val)
				if err != nil {
					return err
				}
				link.HRef = href.String()
			case attr.Namespace == "" && strings.HasPrefix(attr.Key, "data-"):
				link.DataAttrs[attr.Key] = attr.Val
			}
		}
		var text strings.Builder
		_ = visitHTML(node, nil, func(child *html.Node) error {
			if child.Type == html.TextNode {
				text.WriteString(child.Data)
			}
			return nil
		})
		link.Text = text.String()
		links = append(links, link)
		return nil
	}); err != nil {
		return nil, err
	}

	return links, err
}

type PackageLink struct {
	client Client
	Link
}

func (c Client) ListPackages(ctx context.Context) ([]PackageLink, error) {
	c.fillDefaults()
	rawLinks, err := c.getHTML5Index(ctx, c.BaseURL)
	if err != nil {
		return nil, err
	}
	links := make([]PackageLink, 0, len(rawLinks))
	for _, link := range rawLinks {
		links = append(links, PackageLink{
			client: c,
			Link:   link,
		})
	}
	return links, nil
}

type FileLink struct {
	client Client
	Link
}

func (l PackageLink) ListFiles(ctx context.Context) ([]FileLink, error) {
	rawLinks, err := l.client.getHTML5Index(ctx, l.HRef)
	if err != nil {
		return nil, err
	}
	links := make([]FileLink, 0, len(rawLinks))
	for _, link := range rawLinks {
		links = append(links, FileLink{
			client: l.client,
			Link:   link,
		})
	}
	return links, nil
}

func normalize(str string) string {
	return strings.ToLower(regexp.MustCompile("[-_.]+").ReplaceAllLiteralString(str, "-"))
}

func (c Client) ListPackageFiles(ctx context.Context, pkgname string) ([]FileLink, error) {
	// "the only valid characters in a name are the ASCII alphabet, ASCII numbers, `.`, `-`, and
	// `_`."
	for _, char := range pkgname {
		if !(('a' <= char && char <= 'z') ||
			('A' <= char && char <= 'Z') ||
			('0' <= char && char <= '9') ||
			char == '.' ||
			char == '-' ||
			char == '_') {
			return nil, fmt.Errorf("illegal character in pkgname: %q: %s",
				pkgname, strconv.QuoteRuneToASCII(char))
		}
	}

	c.fillDefaults()
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, normalize(pkgname))
	rawLinks, err := c.getHTML5Index(ctx, u.String())
	if err != nil {
		return nil, err
	}
	links := make([]FileLink, 0, len(rawLinks))
	for _, link := range rawLinks {
		if c.Python != nil {
			if reqPy := link.DataAttrs["data-requires-python"]; reqPy != "" {
				ok, err := pep345.HaveRequiredPython(*c.Python, reqPy)
				if err == nil && !ok {
					continue
				}
			}
		}

		links = append(links, FileLink{
			client: c,
			Link:   link,
		})
	}
	return links, nil
}

func (l FileLink) Get(ctx context.Context) ([]byte, error) {
	_, content, err := l.client.get(ctx, l.HRef)
	return content, err
}

var ErrNoSignature = errors.New("no signature")

func (l FileLink) GetSignature(ctx context.Context) ([]byte, error) {
	switch l.DataAttrs["data-gpg-sig"] {
	case "false":
		return nil, ErrNoSignature
	case "true":
		_, content, err := l.client.get(ctx, l.HRef)
		return content, err
	default:
		_, content, err := l.client.get(ctx, l.HRef)
		var httpErr *HTTPError
		if err != nil && errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			err = ErrNoSignature
		}
		return content, err
	}
}
