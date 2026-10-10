package operatorscope_test

import (
	"os"
	"os/exec"
	"testing"
)

func TestOfflineTokenScopeTools(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 required for offline chart scope checks")
	}
	cmd := exec.Command(python, "-m", "unittest", "discover", "-s", "../../scripts/operator-token-scope", "-v")
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("token scope checks failed: %v\n%s", err, out)
	}
}
