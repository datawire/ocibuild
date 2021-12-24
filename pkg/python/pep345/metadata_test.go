package pep345_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ocibuild/pkg/python/pep345"
	"github.com/datawire/ocibuild/pkg/python/pep440"
	"github.com/datawire/ocibuild/pkg/testutil"
)

func parseVersion(t *testing.T, str string) pep440.Version {
	t.Helper()
	ver, err := pep440.ParseVersion(str)
	require.NoError(t, err)
	require.NotNil(t, ver)
	return *ver
}

func TestParseVersionSpecifier(t *testing.T) {
	type TestCase struct {
		Input     string
		OutputVal pep345.VersionSpecifier
		OutputErr string
	}
	testcases := []TestCase{
		{"2.5", pep345.VersionSpecifier{{pep345.CmpOp_EQ, parseVersion(t, "2.5")}}, ""},
		{"==2.5", pep345.VersionSpecifier{{pep345.CmpOp_EQ, parseVersion(t, "2.5")}}, ""},
	}
	t.Parallel()
	for i, tc := range testcases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			spec, err := pep345.ParseVersionSpecifier(tc.Input)
			if tc.OutputErr != "" {
				assert.EqualError(t, err, tc.OutputErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.OutputVal, spec)
			}
		})
	}
}

func TestHaveRequiredPython(t *testing.T) {
	type TestCase struct {
		InputPy   pep440.Version
		InputReq  string
		OutputVal bool
		OutputErr string
	}
	testcases := []TestCase{
		// Check some parse errors
		{parseVersion(t, "2.5a1"), "~=2.5", false, `pep345.ParseVersionSpecifier: pep440.ParseVersion: invalid version: "~=2.5"`},

		// Examples from the spec
		//
		// `Requires-Dist: zope.interface (3.1)`: any version that starts with 3.1,
		// excluding post or pre-releases.
		{parseVersion(t, "3.1"), "3.1", true, ""},
		{parseVersion(t, "3.1.0"), "3.1", true, ""},
		{parseVersion(t, "3.1a1"), "3.1", false, ""},
		{parseVersion(t, "3.1.5a1"), "3.1", false, ""},
		{parseVersion(t, "3.1.5"), "3.1", true, ""},
		{parseVersion(t, "3.2"), "3.1", false, ""},
		{parseVersion(t, "3.1.2post2"), "3.1", false, ""},
		{parseVersion(t, "3.1.2dev3"), "3.1", false, ""},
		// `Requires-Dist: zope.interface (3.1.0)`: any version that starts with 3.1.0,
		// excluding post or pre-releases
		{parseVersion(t, "3.1"), "3.1.0", true, ""},
		{parseVersion(t, "3.1.0"), "3.1.0", true, ""},
		{parseVersion(t, "3.1.0a1"), "3.1", false, ""},
		{parseVersion(t, "3.1.0.5a1"), "3.1", false, ""},
		{parseVersion(t, "3.1.0.5"), "3.1", true, ""},
		{parseVersion(t, "3.1.2"), "3.1", true, ""},
		{parseVersion(t, "3.1.0.2post2"), "3.1", false, ""},
		{parseVersion(t, "3.1.0.2dev3"), "3.1", false, ""},
		// `Requires-Python: 3`: Any Python 3 version, no matter which one, excluding post
		// or pre-releases.
		{parseVersion(t, "3"), "3", true, ""},
		{parseVersion(t, "3.0.0"), "3", true, ""},
		{parseVersion(t, "3.7"), "3", true, ""},
		{parseVersion(t, "3.7a1"), "3", false, ""},
		{parseVersion(t, "3.7-rc3"), "3", false, ""},
		{parseVersion(t, "4.1"), "3", false, ""},
		{parseVersion(t, "2.7"), "3", false, ""},
		// `Requires-Python: >=2.6,<3`: Any version of Python 2.6 or 2.7, including post
		// releases of 2.6, pre and post releases of 2.7. It excludes pre releases of Python
		// 3.
		{parseVersion(t, "2.6rc1"), ">=2.6,<3", false, ""},
		{parseVersion(t, "2.6"), ">=2.6,<3", true, ""},
		{parseVersion(t, "2.6.0"), ">=2.6,<3", true, ""},
		{parseVersion(t, "2.6post1"), ">=2.6,<3", true, ""},
		{parseVersion(t, "2.6.1a1"), ">=2.6,<3", true, ""},
		{parseVersion(t, "2.7"), ">=2.6,<3", true, ""},
		{parseVersion(t, "2.7rc1"), ">=2.6,<3", true, ""},
		{parseVersion(t, "2.7rc1"), ">=2.6,<3", true, ""},
		{parseVersion(t, "3.0rc2"), ">=2.6,<3", false, ""},
		{parseVersion(t, "3.0"), ">=2.6,<3", false, ""},
		// `Requires-Python: 2.6.2`: Equivalent to ">=2.6.2,<2.6.3". So this includes only
		// Python 2.6.2. Of course, if Python was numbered with 4 digits, it would have
		// include all versions of the 2.6.2 series.
		{parseVersion(t, "2.6.2"), "2.6.2", true, ""},
		{parseVersion(t, "2.6.1"), "2.6.2", false, ""},
		{parseVersion(t, "2.6.3"), "2.6.2", false, ""},
		{parseVersion(t, "2.6.2.1"), "2.6.2", true, ""},
		{parseVersion(t, "2.6.2.1rc1"), "2.6.2", false, ""},
		// `Requires-Python: 2.5.0`: Equivalent to ">=2.5.0,<2.5.1".
		{parseVersion(t, "2.5.0"), "2.5.0", true, ""},
		{parseVersion(t, "2.5"), "2.5.0", true, ""},
		{parseVersion(t, "2.5rc1"), "2.5.0", false, ""},
		{parseVersion(t, "2.5.0rc1"), "2.5.0", false, ""},
		{parseVersion(t, "2.5.1rc1"), "2.5.0", false, ""},
		{parseVersion(t, "2.5.1"), "2.5.0", false, ""},
		// `Requires-Dist: zope.interface (3.1,!=3.1.3)`: any version that starts with 3.1,
		// excluding post or pre-releases of 3.1 and excluding any version that starts with
		// "3.1.3". For this particular project, this means: "any version of the 3.1 series
		// but not 3.1.3". This is equivalent to: ">=3.1,!=3.1.3,<3.2".
		{parseVersion(t, "3.1"), "3.1,!=3.1.3", true, ""},
		{parseVersion(t, "3.1.3"), "3.1,!=3.1.3", false, ""},
		{parseVersion(t, "3.1.3.2"), "3.1,!=3.1.3", false, ""},
		{parseVersion(t, "3.1.2rc2"), "3.1,!=3.1.3", false, ""},
		{parseVersion(t, "3.1.2"), "3.1,!=3.1.3", true, ""},

		// Our own testcases
		{parseVersion(t, "3.1dev2"), "==3.1dev2", true, ""},
		{parseVersion(t, "3.1rc1"), "==3.1dev2", false, ""},
		{parseVersion(t, "3.1rc1"), "==3.1rc1", true, ""},
		{parseVersion(t, "3.1post"), ">3.1", true, ""},
		{parseVersion(t, "3.1"), ">=3.1", true, ""},
	}
	t.Parallel()
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.InputPy.String()+"::"+tc.InputReq, func(t *testing.T) {
			t.Parallel()
			have, err := pep345.HaveRequiredPython(tc.InputPy, tc.InputReq)
			if tc.OutputErr != "" {
				assert.EqualError(t, err, tc.OutputErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.OutputVal, have)
			}
		})
	}
}

func TestEquivalentSpecifiers(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"2.6.2", ">=2.6.2,<2.6.3"},
		{"2.5.0", ">=2.5.0,<2.5.1"},
		{"3.1,!=3.1.3", ">=3.1,!=3.1.3,<3.2"},
	}
	staticInputs := []pep440.Version{
		// TODO
	}

	statics := make([][]interface{}, len(staticInputs))
	for i := range statics {
		statics[i] = []interface{}{staticInputs[i]}
	}
	for i, pair := range pairs {
		pair := pair
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			a, err := pep345.ParseVersionSpecifier(pair[0])
			require.NoError(t, err)
			b, err := pep345.ParseVersionSpecifier(pair[1])
			require.NoError(t, err)
			testutil.QuickCheckEqual(t, a.Match, b.Match, testutil.QuickConfig{}, statics...)
		})
	}
}
