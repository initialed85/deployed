// Package fixture guards the tests that need the SSH fixture built from Dockerfile.test, which
// ./env.sh up publishes on localhost:2221. The image is not published anywhere, so lanes that
// cannot start it (CI, any machine without Docker) skip those tests instead of failing on a
// connection refused.
//
// The fixture has to be a separate filesystem from the test process: deploy.go uploads each step
// script to the same path it occupies locally, and upload folders are staged through /tmp, so an
// sshd running in the container that runs the tests itself collides with the local connection
// (scp then fails with "permission denied" on the sticky-bit /tmp).
package fixture

import (
	"net"
	"testing"
	"time"
)

const SSHAddress = "localhost:2221"

func RequireSSH(t *testing.T) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", SSHAddress, time.Second)
	if err != nil {
		t.Skipf("no SSH fixture listening on %s: run ./env.sh up, or docker compose up -d --build ubuntu-ssh-server-2221", SSHAddress)
	}

	_ = conn.Close()
}
