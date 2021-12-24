package python_test

import (
	"fmt"
	"os/exec"
	"testing"
	"testing/quick"

	"github.com/datawire/ocibuild/pkg/python"
)

func TestStatModeString(t *testing.T) {
	t.Parallel()
	fn := func(mode python.StatMode) bool {
		if mode&python.ModeFmt == python.ModeFmtWhiteout {
			// S_IFWHT isn't defined on all platforms (it is on macOS, but not on
			// Linux), and so whether Python's _stat.c:filetype() will return 'w' or '?'
			// for it varies by platform.  But we don't want the Go code's behavior to
			// vary by platform, so just skip ModeFmtWhiteout tests.
			return true
		}

		act := mode.String()

		// #nosec G204 -- it's perfectly safe to insert an integer
		exp, _ := exec.Command("python3", "-c",
			fmt.Sprintf(`import stat; print(stat.filemode(%d), end="")`, mode)).
			Output()

		return act == string(exp)
	}
	if err := quick.Check(fn, nil); err != nil {
		t.Error(err)
	}
}
