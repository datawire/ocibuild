package pep440_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/python/pep440"
)

func TestSort(t *testing.T) {
	t.Parallel()
	testcases := map[string][]string{
		"final-releases-1": []string{
			"0.9",
			"0.9.1",
			"0.9.2",
			"0.9.10",
			"0.9.11",
			"1.0",
			"1.0.1",
			"1.1",
			"2.0",
			"2.0.1",
		},
		"final-releases-2": []string{
			"2012.4",
			"2012.7",
			"2012.10",
			"2013.1",
			"2013.6",
		},
		"version-epochs": []string{
			"1.0",
			"1.1",
			"2.0",
			"2013.10",
			"2014.04",
		},
		"examples-of-compliant-version-schemes-1": []string{
			"0.1",
			"0.2",
			"0.3",
			"1.0",
			"1.1",
		},
		"examples-of-compliant-version-schemes-2": []string{
			"1.1.0",
			"1.1.1",
			"1.1.2",
			"1.2.0",
		},
		"examples-of-compliant-version-schemes-3": []string{
			"0.9",
			"1.0a1",
			"1.0a2",
			"1.0b1",
			"1.0rc1",
			"1.0",
			"1.1a1",
		},
		"examples-of-compliant-version-schemes-4": []string{
			"0.9",
			"1.0.dev1",
			"1.0.dev2",
			"1.0.dev3",
			"1.0.dev4",
			"1.0c1",
			"1.0c2",
			"1.0",
			"1.0.post1",
			"1.1.dev1",
		},
		"examples-of-compliant-version-schemes-5": []string{
			"2012.1",
			"2012.2",
			"2012.3",
			"2012.15",
			"2013.1",
			"2013.2",
		},
		"summary-of-permitted-suffixes-and-relative-ordering": []string{
			"1.0.dev456",
			"1.0a1",
			"1.0a2.dev456",
			"1.0a12.dev456",
			"1.0a12",
			"1.0b1.dev456",
			"1.0b2",
			"1.0b2.post345.dev456",
			"1.0b2.post345",
			"1.0rc1.dev456",
			"1.0rc1",
			"1.0",
			"1.0+abc.5",
			"1.0+abc.7",
			"1.0+5",
			"1.0.post456.dev34",
			"1.0.post456",
			"1.1.dev1",
		},
	}
	for tcName, tcData := range testcases {
		strs := tcData
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()
			vers := make([]*pep440.Version, 0, len(strs))
			exps := make([]string, 0, len(strs))
			for _, str := range strs {
				ver, err := pep440.ParseVersion(str)
				require.NoError(t, err)
				require.NotNil(t, ver)
				vers = append(vers, ver)
				exps = append(exps, ver.String())
			}
			sort.Slice(vers, func(i, j int) bool {
				return vers[i].Cmp(*vers[j]) < 1
			})
			acts := make([]string, 0, len(strs))
			for _, ver := range vers {
				acts = append(acts, ver.String())
			}
			assert.Equal(t, exps, acts)
		})
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()
	type TestCase struct {
		Input      string
		Normalized string // empty for parse error
	}
	testcases := map[string]TestCase{
		"case-sensitivity":                    {"1.1RC1", "1.1rc1"},
		"integer-normalization-1":             {"00", "0"},
		"integer-normalization-2":             {"09000", "9000"},
		"integer-normalization-3":             {"1.0+foo0100", "1.0+foo0100"},
		"pre-release-separators-1":            {"1.1.a1", "1.1a1"},
		"pre-release-separators-2":            {"1.1-a1", "1.1a1"},
		"pre-release-separators-3":            {"1.0a.1", "1.0a1"},
		"pre-release-spelling-1":              {"1.1alpha1", "1.1a1"},
		"pre-release-spelling-2":              {"1.1beta2", "1.1b2"},
		"pre-release-spelling-3":              {"1.1c3", "1.1rc3"},
		"implicit-pre-release-number":         {"1.2a", "1.2a0"},
		"post-release-separators-1":           {"1.2-post2", "1.2.post2"},
		"post-release-separators-2":           {"1.2post2", "1.2.post2"},
		"post-release-separators-3":           {"1.2.post.2", "1.2.post2"},
		"post-release-spelling":               {"1.0-r4", "1.0.post4"},
		"implicit-post-release-number":        {"1.2.post", "1.2.post0"},
		"implicit-post-releases-1":            {"1.0-1", "1.0.post1"},
		"implicit-post-releases-2":            {"1.0-", ""},
		"implicit-post-releases-extra":        {"1.0_1", ""},
		"development-release-separators-1":    {"1.2-dev2", "1.2.dev2"},
		"development-release-separators-2":    {"1.2dev2", "1.2.dev2"},
		"implicit-development-release-number": {"1.2.dev", "1.2.dev0"},
		"local-version-segments":              {"1.0+ubuntu-1", "1.0+ubuntu.1"},
		"preceding-v-character":               {"v1.0", "1.0"},
		"leading-and-trailing-whitespace":     {"1.0\n", "1.0"},
	}
	for tcName, tcData := range testcases {
		tcData := tcData
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()
			t.Logf("input: %q", tcData.Input)
			ver, err := pep440.ParseVersion(tcData.Input)
			if tcData.Normalized == "" {
				assert.Error(t, err)
				assert.Nil(t, ver)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, ver)
				assert.Equal(t, tcData.Normalized, ver.String())
			}
		})
	}
}
