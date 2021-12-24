package direct_url

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONDumps(t *testing.T) {
	type testcase struct {
		Input  interface{}
		Output string
	}
	testcases := []testcase{
		//nolint:lll // long literals
		{
			Input: DirectURL{
				URL:         "file:///run/user/1000/tmpdir/TestPIPFlask-1.1.2-py2.py3-none-any.whl2100032774/001/Flask-1.1.2-py2.py3-none-any.whl",
				ArchiveInfo: &ArchiveInfo{},
			},
			Output: `{"archive_info": {}, "url": "file:///run/user/1000/tmpdir/TestPIPFlask-1.1.2-py2.py3-none-any.whl2100032774/001/Flask-1.1.2-py2.py3-none-any.whl"}`,
		},
	}
	for i, tc := range testcases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out, err := jsonDumps(tc.Input)
			assert.NoError(t, err)
			assert.Equal(t, tc.Output, string(out))
		})
	}
}
