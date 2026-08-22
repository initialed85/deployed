package connection

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSH(t *testing.T) {
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
		_, err := OpenSSH("__local__", 0, "${USER}", "${PASS}")
		require.Error(t, err)

		_, err = OpenSSH("__local__", 22, "${USER}", "${PASS}")
		require.Error(t, err)

		c, err := OpenSSH("localhost", 2221, "user2", "Password2")
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

		stdout, stderr, err = c.RunCommand("cat /etc/shadow")
		require.Error(t, err)
		require.Empty(t, stdout)
		require.NotEmpty(t, stderr)

		stdout, stderr, err = c.RunCommandWithSudo("cat /etc/shadow")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Empty(t, stderr)

		_, _, _ = c.RunCommandWithSudo("rm -fv /var/log/weird-file.txt")
		err = c.Upload("/tmp/some-file.txt", "/var/log/weird-file.txt")
		require.Error(t, err)

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

		stdout, stderr, err = c.RunCommand("cd /srv/")
		require.NoError(t, err)

		stdout, stderr, err = c.RunCommand("pwd")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Equal(t, fmt.Sprintf("/home/%s\n", c.GetUsername()), stdout)

		stdout, stderr, err = c.RunCommand("echo ${PWD}")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Equal(t, fmt.Sprintf("/home/%s\n", c.GetUsername()), stdout)

		err = c.Download("/var/log/weird-file.txt", "/tmp/weird-file.txt")
		require.NoError(t, err)

		b, err := os.ReadFile("/tmp/weird-file.txt")
		require.NoError(t, err)
		require.Len(t, b, 10_000_000)

		err = c.Download("/etc/shadow", "/tmp/etc-shadow.txt")
		require.Error(t, err)

		err = c.DownloadWithSudo("/etc/shadow", "/tmp/etc-shadow.txt")
		require.NoError(t, err)

		err = c.Download("/etc", "/tmp/etc")
		require.Error(t, err)

		// TODO(initialed85): fix the test harness requiring sudo
		_, err = exec.Command("sh", "-c", fmt.Sprintf("sudo -n chown -fR '%s:%s' /tmp/etc", os.Getenv("USER"), os.Getenv("USER"))).Output()
		require.NoError(t, err)

		err = c.DownloadWithSudo("/etc", "/tmp/etc")
		require.NoError(t, err)

		err = c.DownloadWithSudo("/etc", "/tmp/etc")
		require.NoError(t, err)

		b, err = os.ReadFile("/tmp/etc/hostname")
		require.NoError(t, err)
		require.NotEmpty(t, b)
	})
}
