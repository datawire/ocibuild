package direct_url //nolint:testpackage // testing an internal thing

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONDumps(t *testing.T) {
	t.Parallel()
	type testcase struct {
		Input  interface{}
		Output string
	}
	testcases := []testcase{
		//nolint:lll // long literals
		{
			Input: DirectURL{ //nolint:exhaustivestruct
				URL:         "file:///run/user/1000/tmpdir/TestPIPFlask-1.1.2-py2.py3-none-any.whl2100032774/001/Flask-1.1.2-py2.py3-none-any.whl",
				ArchiveInfo: &ArchiveInfo{}, //nolint:exhaustivestruct
			},
			Output: `{"archive_info": {}, "url": "file:///run/user/1000/tmpdir/TestPIPFlask-1.1.2-py2.py3-none-any.whl2100032774/001/Flask-1.1.2-py2.py3-none-any.whl"}`,
		},
	}
	for i, tc := range testcases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			out, err := jsonDumps(tc.Input)
			assert.NoError(t, err)
			assert.Equal(t, tc.Output, string(out))
		})
	}
}
