package testutil

import (
	"archive/tar"
	"fmt"
	"io"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/davecgh/go-spew/spew"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pmezard/go-difflib/difflib"
)

func DumpLayerFull(layer ociv1.Layer) (str string, err error) {
	maybeSetErr := func(_err error) {
		if _err != nil && err == nil {
			str = ""
			err = _err
		}
	}

	var spewConfig = spew.ConfigState{
		Indent:                  "  ",
		DisableMethods:          true,
		DisableCapacities:       true,
		DisablePointerAddresses: true,
		SortKeys:                true,
	}

	ret := new(strings.Builder)

	layerReader, _err := layer.Uncompressed()
	if _err != nil {
		return "", err
	}
	defer func() {
		maybeSetErr(layerReader.Close())
	}()

	tarReader := tar.NewReader(layerReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(ret, "tarHeader = %s", spewConfig.Sdump(header)); err != nil {
			return "", err
		}

		content, err := io.ReadAll(tarReader)
		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(ret, "tarContent =%s", spewConfig.Sdump(content)); err != nil {
			return "", err
		}
	}

	rest, err := io.ReadAll(layerReader)
	if err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(ret, "tail =\n%s", spewConfig.Sdump(rest)); err != nil {
		return "", err
	}

	return ret.String(), nil
}

func DumpLayerListing(layer ociv1.Layer) (str string, err error) {
	maybeSetErr := func(_err error) {
		if _err != nil && err == nil {
			str = ""
			err = _err
		}
	}

	ret := new(strings.Builder)

	layerReader, _err := layer.Uncompressed()
	if _err != nil {
		return "", err
	}
	defer func() {
		maybeSetErr(layerReader.Close())
	}()

	table := tabwriter.NewWriter(
		ret, // output
		0,   // minwidth
		1,   // tabwidth
		1,   // padding
		' ', // padchar
		0)   // flags
	tarReader := tar.NewReader(layerReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintln(table, strings.Join([]string{
			"",
			header.FileInfo().Mode().String(),
			fmt.Sprintf("%d=%q", header.Uid, header.Uname),
			fmt.Sprintf("%d=%q", header.Gid, header.Gname),
			fmt.Sprintf("% 10d", header.Size),
			header.Name,
		}, "\t")); err != nil {
			return "", err
		}

		if _, err := io.ReadAll(tarReader); err != nil {
			return "", err
		}
	}
	if err := table.Flush(); err != nil {
		return "", err
	}

	return ret.String(), nil
}

func AssertEqualLayers(t *testing.T, exp, act ociv1.Layer) bool {
	t.Helper()

	// First just compare the listings, in order to "fail fast" and give more readable output.
	expStr, err := DumpLayerListing(exp)
	if err != nil {
		t.Errorf("error dumping expected layer listing: %v", err)
		return false
	}
	actStr, err := DumpLayerListing(act)
	if err != nil {
		t.Errorf("error dumping actual layer listing: %v", err)
		return false
	}
	if expStr != actStr {
		diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(expStr),
			B:        difflib.SplitLines(actStr),
			FromFile: "Expected",
			FromDate: "",
			ToFile:   "Actual",
			ToDate:   "",
			Context:  1,
		})
		t.Errorf("Listing diff:\n%s", diff)
		return false
	}

	// OK, that passed, now dow a comre comprehensive diff.
	expStr, err = DumpLayerFull(exp)
	if err != nil {
		t.Errorf("error dumping expected layer: %v", err)
		return false
	}
	actStr, err = DumpLayerFull(act)
	if err != nil {
		t.Errorf("error dumping actual layer: %v", err)
		return false
	}
	if expStr != actStr {
		diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(expStr),
			B:        difflib.SplitLines(actStr),
			FromFile: "Expected",
			FromDate: "",
			ToFile:   "Actual",
			ToDate:   "",
			Context:  1,
		})
		t.Errorf("Full diff:\n%s", diff)
		return false
	}

	return true
}
