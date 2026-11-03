package connection

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocal(t *testing.T) {
	err := os.WriteFile("/tmp/some-file.txt", []byte("this is a file\n"), 0o644)
	require.NoError(t, err)

	b := make([]byte, 10_000_000)
	for i := range b {
		b[i] = 'a'
	}

	err = os.WriteFile("/tmp/other-file.bin", b, 0o644)
	require.NoError(t, err)

	err = os.MkdirAll("/tmp/some-folder", 0o777)
	require.NoError(t, err)

	err = os.MkdirAll("/tmp/other-folder", 0o777)
	require.NoError(t, err)

	_, err = exec.Command("cp", "-fv", "/tmp/some-file.txt", "/tmp/some-folder/").CombinedOutput()
	require.NoError(t, err)

	_, err = exec.Command("cp", "-fv", "/tmp/some-file.txt", "/tmp/some-folder/other-file.txt").CombinedOutput()
	require.NoError(t, err)

	_, err = exec.Command("cp", "-fv", "/tmp/other-file.bin", "/tmp/some-folder/").CombinedOutput()
	require.NoError(t, err)

	_, err = exec.Command("cp", "-fv", "/tmp/other-file.bin", "/tmp/some-folder/some-file.bin").CombinedOutput()
	require.NoError(t, err)

	t.Run("LongLivedConnection", func(t *testing.T) {
		_, err := OpenLocal("some.host.org", 22, "${USER}", "${PASS}")
		require.Error(t, err)

		_, err = OpenLocal("__local__", 22, "${USER}", "${PASS}")
		require.Error(t, err)

		c, err := OpenLocal("__local__", 0, "${USER}", "${PASS}")
		require.NoError(t, err)

		defer c.Close()

		stdout, stderr, err := c.RunCommand("free -h")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		stdout, stderr, err = c.RunCommand("lscpu")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		// A local connection runs as whoever invoked it, so the assertions below only hold for an unprivileged
		// one. The live lane runs its tests as root because root is what owns libvirtd, and there a forbidden file
		// is a property of the harness rather than something deployed could get wrong.
		rootRunner := os.Getuid() == 0

		if !rootRunner {
			stdout, stderr, err = c.RunCommand("cat /etc/shadow")
			require.Error(t, err)
			require.Empty(t, stdout)
			require.NotEmpty(t, stderr)
		}

		stdout, stderr, err = c.RunCommandWithSudo("cat /etc/shadow")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		_, _, _ = c.RunCommandWithSudo("rm -fv /var/log/weird-file.txt")
		if !rootRunner {
			err = c.Upload("/tmp/some-file.txt", "/var/log/weird-file.txt")
			require.Error(t, err)
		}

		err = c.UploadWithSudo("/tmp/some-file.txt", "/var/log/weird-file.txt")
		require.NoError(t, err)

		err = c.UploadWithSudo("/tmp/other-file.bin", "/var/log/weird-file.txt")
		require.NoError(t, err)

		err = c.Upload("/tmp/some-folder", fmt.Sprintf("/home/%s/some-folder", c.GetUsername()))
		require.NoError(t, err)

		err = c.UploadWithSudo("/tmp/some-folder", "/var/log/weird-folder")
		require.NoError(t, err)

		stdout, stderr, err = c.RunCommandWithSudo("du -sh /var/log/weird-file.txt")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		_, _, err = c.RunCommand("cd /srv/")
		require.NoError(t, err)

		// A fresh local connection starts in the invoking user's home directory, which is what the connection
		// resolves HOME to. Reading it as /home plus $USER held until the live lane ran the suite as root: root's
		// home is /root and a container leaves USER unset, so the expectation was neither of the two.
		home := fmt.Sprintf("%s\n", os.Getenv("HOME"))

		stdout, _, err = c.RunCommand("pwd")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Equal(t, home, stdout)

		stdout, _, err = c.RunCommand("echo ${PWD}")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Equal(t, home, stdout)

		// more output than the pipe buffer holds must still arrive in full
		stdout, _, err = c.RunCommand("seq 1 5000")
		require.NoError(t, err)
		require.Equal(t, 5000, strings.Count(stdout, "\n"))

		err = c.Download("/var/log/weird-file.txt", "/tmp/weird-file.txt")
		require.NoError(t, err)

		b, err := os.ReadFile("/tmp/weird-file.txt")
		require.NoError(t, err)
		require.Len(t, b, 10_000_000)

		// a single-file download has to create the local parent folder, as folder downloads already do
		err = c.Download("/var/log/weird-file.txt", "/tmp/weird-file-nested-dir/weird-file.txt")
		require.NoError(t, err)

		b, err = os.ReadFile("/tmp/weird-file-nested-dir/weird-file.txt")
		require.NoError(t, err)
		require.Len(t, b, 10_000_000)

		// TODO(initialed85): idk, platform variance I think
		err = c.Download("/etc/shadow", "/tmp/etc-shadow.txt")
		if err != nil {
			require.Error(t, err)

			err = c.DownloadWithSudo("/etc/shadow", "/tmp/etc-shadow.txt")
			require.NoError(t, err)
		}

		if !rootRunner {
			err = c.Download("/etc", "/tmp/etc")
			require.Error(t, err)
		}

		err = c.DownloadWithSudo("/etc", "/tmp/etc")
		require.NoError(t, err)

		b, err = os.ReadFile("/tmp/etc/hostname")
		require.NoError(t, err)
		require.NotEmpty(t, b)
	})
}

func TestLocalScriptSeesTheUsernameTheConnectionResolved(t *testing.T) {
	// Uploads and downloads resolve ${USER} against the connection's own username.  A script is only a
	// shell, so the shell has to be told the same thing - and a container ships no USER at all, which is
	// where the two used to disagree.
	t.Setenv("USER", "")

	c, err := OpenLocal("__local__", 0, "${USER}", "${PASS}")
	require.NoError(t, err)

	require.NotEmpty(t, c.GetUsername())

	stdout, _, err := c.RunCommand("echo ${USER}")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s\n", c.GetUsername()), stdout)
}
