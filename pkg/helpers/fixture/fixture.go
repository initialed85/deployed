// Package fixture guards the tests that need the SSH fixture built from Dockerfile.test, which
// ./env.sh up publishes on localhost:2221. The image is not published anywhere, so a lane that cannot start
// it says so and skips those tests instead of failing on a connection refused. A lane that can reach a
// fixture somewhere else names it in DEPLOYED_SSH_FIXTURE, and then a fixture that is not there is a failure.
//
// The fixture has to be a separate filesystem from the test process: deploy.go uploads each step
// script to the same path it occupies locally, and upload folders are staged through /tmp, so an
// sshd running in the container that runs the tests itself collides with the local connection
// (scp then fails with "permission denied" on the sticky-bit /tmp).
package fixture

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"
)

// SSHAddress is where the fixture listens. DEPLOYED_SSH_FIXTURE overrides it for a lane that reaches the
// fixture somewhere other than localhost:2221 - the unit CI lane runs it as a service pod, where the host is
// the service name and the port is the container's own 2222, with no published port in between.
var SSHAddress = func() string {
	if v := os.Getenv("DEPLOYED_SSH_FIXTURE"); v != "" {
		return v
	}

	return "localhost:2221"
}()

// HostPort is SSHAddress in the shape connection.OpenSSH wants. A bad DEPLOYED_SSH_FIXTURE is a lane
// misconfiguration and not a test failure, so it dies here with the offending value in the message.
func HostPort() (string, int) {
	host, port, err := net.SplitHostPort(SSHAddress)
	if err != nil {
		panic(fmt.Sprintf("DEPLOYED_SSH_FIXTURE %q: %v", SSHAddress, err))
	}

	n, err := strconv.Atoi(port)
	if err != nil {
		panic(fmt.Sprintf("DEPLOYED_SSH_FIXTURE %q: %v", SSHAddress, err))
	}

	return host, n
}

func RequireSSH(t *testing.T) {
	t.Helper()

	// A lane that names a fixture has promised one. Without DEPLOYED_SSH_FIXTURE the fixture is optional and a
	// missing one is a machine without Docker, so the test says so and skips. With it set, the address is a
	// deliberate claim about the environment and nothing answering is a broken lane, not a skipped test.
	configured := os.Getenv("DEPLOYED_SSH_FIXTURE") != ""

	conn, err := net.DialTimeout("tcp", SSHAddress, time.Second)
	if err != nil {
		if configured {
			t.Fatalf("DEPLOYED_SSH_FIXTURE puts the fixture on %s and nothing answered: %v", SSHAddress, err)
		}

		t.Skipf("no SSH fixture listening on %s: run ./env.sh up, or docker compose up -d --build ubuntu-ssh-server-2221", SSHAddress)
	}

	_ = conn.Close()
}
