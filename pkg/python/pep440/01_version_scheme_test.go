package pep440_test

import (
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/python/pep440"
	"github.com/datawire/ocibuild/pkg/testutil"
)

func TestSort(t *testing.T) {
	t.Parallel()
	testcases := map[string][]string{
		// from the spec
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
		"pre-releases": []string{
			"4.3a2",  // Alpha release
			"4.3b2",  // Beta release
			"4.3rc2", // Release Candidate
			"4.3",    // Final release
		},
		"post-releases": []string{
			"4.3a2.post1",  // Post-release of an alpha release
			"4.3b2.post1",  // Post-release of a beta release
			"4.3rc2.post1", // Post-release of a release candidate
		},
		"developmental-releases": []string{
			"4.3a2.dev1",     // Developmental release of an alpha release
			"4.3b2.dev1",     // Developmental release of a beta release
			"4.3rc2.dev1",    // Developmental release of a release candidate
			"4.3.post2.dev1", // Developmental release of a post-release
		},
		"version-epochs-1": []string{
			"1.0",
			"1.1",
			"2.0",
			"2013.10",
			"2014.04",
		},
		"version-epochs-2": []string{
			"2013.10",
			"2014.04",
			"1!1.0",
			"1!1.1",
			"1!2.0",
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
		// our own
		"local-segment": []string{
			"1.0",
			"1.0+a",
			"1.0+b",
			"1.0+bar",
			"1.0+c",
			"1.0+d",
			"1.0+e",
			"1.0+f",
			"1.0+z",
			"1.0+0",
			"1.0+0.z",
			"1.0+0.0",
			"1.0+0.0.0",
			"1.0+1",
			"1.0+2",
			"1.0+10",
			"1.1",
		},
	}
	for tcName, tcData := range testcases {
		strs := tcData
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()
			rand := rand.New(rand.NewSource(time.Now().UnixNano()))

			vers := make([]*pep440.Version, 0, len(strs))
			exps := make([]string, 0, len(strs))
			for _, str := range strs {
				ver, err := pep440.ParseVersion(str)
				require.NoError(t, err)
				require.NotNil(t, ver)
				vers = append(vers, ver)
				exps = append(exps, ver.String())
			}

			// shuffle the list so that `sort` has something to do.
			rand.Shuffle(len(vers), func(i, j int) {
				vers[i], vers[j] = vers[j], vers[i]
			})

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
				if len(ver.Local) == 0 {
					assert.Equal(t, tcData.Normalized, ver.PublicVersion.String())
				}
			}
		})
	}
}

func TestEquality(t *testing.T) {
	t.Parallel()

	staticInputs := []pep440.Version{
		// TODO
	}

	testutil.QuickCheck(t,
		// test function
		func(ver1 pep440.Version) bool {
			_ver2, err := pep440.ParseVersion(ver1.String())
			if err != nil || _ver2 == nil {
				return false
			}
			ver2 := *_ver2
			return (ver1.Cmp(ver2) == 0) && (ver2.Cmp(ver1) == 0)
		},
		// dynamic inputs
		testutil.QuickConfig{},
		// static inputs
		func() [][]interface{} {
			ret := make([][]interface{}, len(staticInputs))
			for i := range ret {
				ret[i] = []interface{}{staticInputs[i]}
			}
			return ret
		}()...)
}

func TestSymmetry(t *testing.T) {
	t.Parallel()
	const (
		partNone = iota
		partEpoch
		partRel
		partPre
		partPost
		partDev
		partLocal
	)
	names := []string{
		"none",
		"epoch",
		"rel",
		"pre",
		"post",
		"dev",
		"local",
	}
	staticInputs := [][2]pep440.Version{
		{mustParseVersion(t, "1.0+1.0"), mustParseVersion(t, "1.0+1.0.0")},
		{mustParseVersion(t, "1.0+1.foo"), mustParseVersion(t, "1.0+1.bar")},
	}

	statics := make([][]interface{}, len(staticInputs))
	for i := range statics {
		statics[i] = []interface{}{
			staticInputs[i][0],
			staticInputs[i][1],
		}
	}

	for lockdown := 0; lockdown <= partLocal; lockdown++ {
		lockdown := lockdown
		t.Run("lockdown-"+names[lockdown], func(t *testing.T) {
			t.Parallel()
			testutil.QuickCheck(t,
				// test function
				func(ver1, ver2 pep440.Version) bool {
					if lockdown >= partEpoch {
						ver2.Epoch = ver1.Epoch
					}
					if lockdown >= partRel {
						ver2.Release = ver1.Release
					}
					if lockdown >= partPre {
						ver2.Pre = ver1.Pre
					}
					if lockdown >= partPost {
						ver2.Post = ver1.Post
					}
					if lockdown >= partDev {
						ver2.Dev = ver1.Dev
					}
					if lockdown >= partLocal {
						ver2.Local = ver1.Local
					}
					ret := ver1.Cmp(ver2) == -ver2.Cmp(ver1)
					if lockdown == partLocal {
						ret = ret && ver1.Cmp(ver2) == 0 && ver2.Cmp(ver1) == 0
					}
					if !ret {
						t.Logf("failing:\n\tver1=%s\n\tver2=%s\n\tver1.Cmp(ver2)=%v\n\tver2.Cmp(ver1)=%v",
							ver1, ver2,
							ver1.Cmp(ver2), ver2.Cmp(ver1))
					}
					return ret
				},
				// dynamic inputs
				testutil.QuickConfig{},
				// static inputs
				statics...)
		})
	}
}

func TestUtilMethods(t *testing.T) {
	t.Parallel()
	type TestCase struct {
		Input pep440.Version

		// shared
		Major        int
		Minor        int
		Micro        int
		IsPreRelease bool

		// might be different

		LocalString  string
		LocalIsFinal bool

		PublicString  string
		PublicIsFinal bool
	}
	testcases := []TestCase{
		{mustParseVersion(t, "1           "), 1, 0, 0, false /*local*/, "1           ", true /**public*/, "1           ", true},
		{mustParseVersion(t, "1+par       "), 1, 0, 0, false /*local*/, "1+par       ", false /*public*/, "1           ", true},
		{mustParseVersion(t, "1.2         "), 1, 2, 0, false /*local*/, "1.2         ", true /**public*/, "1.2         ", true},
		{mustParseVersion(t, "1.2.3       "), 1, 2, 3, false /*local*/, "1.2.3       ", true /**public*/, "1.2.3       ", true},
		{mustParseVersion(t, "1.2rc2      "), 1, 2, 0, true /**local*/, "1.2rc2      ", false /*public*/, "1.2rc2      ", false},
		{mustParseVersion(t, "1.2rc2.post3"), 1, 2, 0, true /**local*/, "1.2rc2.post3", false /*public*/, "1.2rc2.post3", false},
		{mustParseVersion(t, "1.2rc2+par  "), 1, 2, 0, true /**local*/, "1.2rc2+par  ", false /*public*/, "1.2rc2      ", false},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.Input.String(), func(t *testing.T) {
			assert.Equal(t, tc.Input.Major(), tc.Major, "Major")
			assert.Equal(t, tc.Input.Minor(), tc.Minor, "Minor")
			assert.Equal(t, tc.Input.Micro(), tc.Micro, "Micro")
			assert.Equal(t, tc.Input.IsPreRelease(), tc.IsPreRelease, "IsPreRelease")

			assert.Equal(t, tc.Input.String(), strings.TrimSpace(tc.LocalString), "LocalVersion.String")
			assert.Equal(t, tc.Input.IsFinal(), tc.LocalIsFinal, "LocalVersion.IsFinal")

			assert.Equal(t, tc.Input.PublicVersion.String(), strings.TrimSpace(tc.PublicString), "PublicVersion.String")
			assert.Equal(t, tc.Input.PublicVersion.IsFinal(), tc.PublicIsFinal, "PublicVersion.IsFinal")
		})
	}
}
