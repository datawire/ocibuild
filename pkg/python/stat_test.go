package python_test

import (
	"fmt"
	"os/exec"
	"testing"
	"testing/quick"

	"github.com/datawire/ocibuild/pkg/python"
)

func TestStatModeString(t *testing.T) {
	fn := func(m python.StatMode) bool {
		act := m.String()
		exp, _ := exec.Command("python3", "-c",
			fmt.Sprintf(`import stat; print(stat.filemode(%d), end="")`, m)).
			Output()
		return act == string(exp)
	}
	if err := quick.Check(fn, nil); err != nil {
		t.Error(err)
	}
}
