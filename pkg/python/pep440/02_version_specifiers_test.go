package pep440_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/datawire/ocibuild/pkg/python/pep440"
	"github.com/datawire/ocibuild/pkg/testutil"
)

func TestParseSpecifier(t *testing.T) {
	t.Parallel()
	testcases := map[string]struct {
		InStr  string
		OutVal pep440.Specifier
		OutErr string
	}{
		"empty":       {"", pep440.Specifier{}, ""},
		"whitespace":  {"  ", pep440.Specifier{}, ""},
		"emptycommas": {", ,", pep440.Specifier{}, ""},
		"eq":          {"==1.0", pep440.Specifier{{pep440.CmpOp_StrictMatch, mustParseVersion(t, "1.0")}}, ""},
		"missing-op":  {"1.0", nil, `pep440.ParseSpecifier: invalid comparison operator: "1.0"`},
		"1seg-ok":     {"==1", pep440.Specifier{{pep440.CmpOp_StrictMatch, mustParseVersion(t, "1")}}, ""},
		"1seg-bad":    {"~=1", nil, `pep440.ParseSpecifier: at least 2 release segments required in ~= specifier clauses`},
		"bad-dev":     {"==1.0dev.*", nil, `pep440.ParseSpecifier: dev-part not permitted in prefix == specifier clauses`},
		"bad-loc":     {"==1.0+loc.*", nil, `pep440.ParseSpecifier: local-part not permitted in prefix == specifier clauses`},
	}
	for tcName, tc := range testcases {
		tc := tc
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()
			val, err := pep440.ParseSpecifier(tc.InStr)
			assert.Equal(t, tc.OutVal, val)
			if tc.OutErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.OutErr)
			}
		})
	}
}

func TestEquivalentSpecifiers(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"~= 2.2", ">= 2.2, == 2.*"},
		{"~= 1.4.5", ">= 1.4.5, == 1.4.*"},
		{"~= 2.2.post3", ">= 2.2.post3, == 2.*"},
		{"~= 1.4.5a4", ">= 1.4.5a4, == 1.4.*"},
		{"~= 2.2.0", ">= 2.2.0, == 2.2.*"},
		{"~= 1.4.5.0", ">= 1.4.5.0, == 1.4.5.*"},
	}
	staticInputs := []pep440.Version{
		pep440.LocalVersion{
			PublicVersion: pep440.PublicVersion{Epoch: 0, Release: []int{2, 2654, 2662, 1281, 1226}, Pre: &pep440.PreRelease{L: "rc", N: 2647}, Post: nil, Dev: nil},
			Local:         nil,
		},
		pep440.LocalVersion{
			PublicVersion: pep440.PublicVersion{Epoch: 0, Release: []int{2, 418, 849}, Pre: nil, Post: intPtr(2328), Dev: intPtr(109)},
			Local:         []intstr.IntOrString{intstr.IntOrString{Type: 0, IntVal: 830, StrVal: ""}, intstr.IntOrString{Type: 1, IntVal: 0, StrVal: "je4kz"}, intstr.IntOrString{Type: 0, IntVal: 2083, StrVal: ""}, intstr.IntOrString{Type: 0, IntVal: 2694, StrVal: ""}, intstr.IntOrString{Type: 0, IntVal: 1127, StrVal: ""}, intstr.IntOrString{Type: 0, IntVal: 142, StrVal: ""}, intstr.IntOrString{Type: 0, IntVal: 1122, StrVal: ""}, intstr.IntOrString{Type: 0, IntVal: 2676, StrVal: ""}, intstr.IntOrString{Type: 1, IntVal: 0, StrVal: "iyf3f9poj7"}},
		},
	}

	statics := make([][]interface{}, len(staticInputs))
	for i := range statics {
		statics[i] = []interface{}{staticInputs[i]}
	}
	for i, pair := range pairs {
		pair := pair
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			a, err := pep440.ParseSpecifier(pair[0])
			require.NoError(t, err)
			b, err := pep440.ParseSpecifier(pair[1])
			require.NoError(t, err)
			testutil.QuickCheckEqual(t, a.Match, b.Match, testutil.QuickConfig{}, statics...)
		})
	}
}

func TestSpecifiers(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		InVer    string
		InSpec   string
		OutMatch bool
	}{
		// from the spec
		{"1.1.post1", "== 1.1", false},
		{"1.1.post1", "== 1.1.post1", true},
		{"1.1.post1", "== 1.1.*", true},

		{"1.1a1", "== 1.1", false},
		{"1.1a1", "== 1.1a1", true},
		{"1.1a1", "== 1.1.*", true},

		{"1.1", "== 1.1", true},
		{"1.1", "== 1.1.0", true},
		{"1.1", "== 1.1.dev1", false},
		{"1.1", "== 1.1a1", false},
		{"1.1", "== 1.1.post1", false},
		{"1.1", "== 1.1.*", true},

		{"1.1.post1", "!= 1.1", true},
		{"1.1.post1", "!= 1.1.post1", false},
		{"1.1.post1", "!= 1.1.*", false},

		// from references
		{"1.7.2", "> 1.7", true},
		{"1.7a1", "< 1.7", true},

		// our own
		{"1!1.2", "== 1.*", false},
		{"1.2", "== 1.*", true},
		{"1.2", "== 1!1.*", false},
		{"1.0", "<= 2.0", true},
		{"1.1rc0", "== 1.1rc.*", true},
		{"1.1rc1", "== 1.1rc.*", false},
		{"1.1post0", "== 1.1post.*", true},
		{"1.1post1", "== 1.1post.*", false},
		{"1rc1", "", true},
	}
	for i, tc := range testcases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			t.Logf("checking: (%s %s) => %v", tc.InVer, tc.InSpec, tc.OutMatch)

			ver, err := pep440.ParseVersion(tc.InVer)
			require.NoError(t, err)
			require.NotNil(t, ver)

			spec, err := pep440.ParseSpecifier(tc.InSpec)
			require.NoError(t, err)

			require.Equal(t, tc.OutMatch, spec.Match(*ver))
		})
	}
}
