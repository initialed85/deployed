// The live lab is one shared set of hosts and go runs test packages in parallel, so TestDeploy
// (pkg/deploy) and TestRollout (pkg/rollout) each run a k3s install on 192.168.56.11 at the same time. The
// k3s installer opens with systemd_disable, which is rm -f on /etc/systemd/system/k3s.service.env, so one
// installer removes the file the other has just created and then dies on the chmod that follows with
// "cannot access ... No such file or directory". Every attempt one failure the live CI lane has had came
// that way, and every one of them was on 192.168.56.11.
//
// The lock covers the tier rather than a host, because the two tests also install agents on the same three
// hosts. It blocks instead of failing, which is the outcome the tier wants anyway: whoever arrives second
// finds k3s already installed and no-ops.
package fixture

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func RequireLiveTier(t *testing.T) {
	t.Helper()

	path := filepath.Join(os.TempDir(), "deployed-live-tier.lock")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("cannot open live tier lock: %v", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatalf("cannot lock live tier: %v", err)
	}

	t.Cleanup(func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	})
}
