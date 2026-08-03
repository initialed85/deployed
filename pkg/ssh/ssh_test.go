package ssh

import (
	"fmt"
	"os"
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

	t.Run("RunCommandOnHostWithNoSudoAtAll", func(t *testing.T) {
		stdout, stderr, err := RunCommand("localhost", 2221, "user1", "Password1", "echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		require.Empty(t, stderr)
		require.Equal(t, "Hello, world.\n", stdout)
		fmt.Println()

		stdout, stderr, err = RunCommandWithSudo("localhost", 2221, "user1", "Password1", "echo 'Hello, world.'")
		require.Error(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		fmt.Println()

		err = TransferFile("localhost", 2221, "user1", "Password1", "/tmp/some-file.txt", "/home/user1/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = TransferFileWithSudo("localhost", 2221, "user1", "Password1", "/tmp/some-file.txt", "/etc/some-file.txt")
		require.Error(t, err)
		fmt.Println()

		err = TransferFileWithSudo("localhost", 2221, "user1", "Password1", "/tmp/other-file.bin", "/etc/other-file.bin")
		require.Error(t, err)
		fmt.Println()
	})

	t.Run("RunCommandOnHostWithSudo", func(t *testing.T) {
		stdout, stderr, err := RunCommand("localhost", 2221, "user2", "Password2", "echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		require.Empty(t, stderr)
		require.Equal(t, "Hello, world.\n", stdout)
		fmt.Println()

		stdout, stderr, err = RunCommandWithSudo("localhost", 2221, "user2", "Password2", "echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		fmt.Println()

		err = TransferFile("localhost", 2221, "user2", "Password2", "/tmp/some-file.txt", "/home/user2/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = TransferFileWithSudo("localhost", 2221, "user2", "Password2", "/tmp/some-file.txt", "/etc/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = TransferFileWithSudo("localhost", 2221, "user2", "Password2", "/tmp/other-file.bin", "/etc/other-file.bin")
		require.NoError(t, err)
		fmt.Println()
	})

	t.Run("RunCommandOnHostWithSudoNoPassword", func(t *testing.T) {
		stdout, stderr, err := RunCommand("localhost", 2221, "user3", "Password3", "echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		require.Empty(t, stderr)
		require.Equal(t, "Hello, world.\n", stdout)
		fmt.Println()

		stdout, stderr, err = RunCommandWithSudo("localhost", 2221, "user3", "Password3", "echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		fmt.Println()

		err = TransferFile("localhost", 2221, "user3", "Password3", "/tmp/some-file.txt", "/home/user3/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = TransferFileWithSudo("localhost", 2221, "user3", "Password3", "/tmp/some-file.txt", "/etc/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = TransferFileWithSudo("localhost", 2221, "user3", "Password3", "/tmp/other-file.bin", "/etc/other-file.bin")
		require.NoError(t, err)
		fmt.Println()
	})

	t.Run("RunCommandSudoNotNeeded", func(t *testing.T) {
		stdout, stderr, err := RunCommand("localhost", 2229, "root", "Password9", "echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		require.Empty(t, stderr)
		require.Equal(t, "Hello, world.\n", stdout)
		fmt.Println()

		stdout, stderr, err = RunCommandWithSudo("localhost", 2229, "root", "Password9", "echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		fmt.Println()

		err = TransferFile("localhost", 2229, "root", "Password9", "/tmp/some-file.txt", "/root/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = TransferFileWithSudo("localhost", 2229, "root", "Password9", "/tmp/some-file.txt", "/etc/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = TransferFileWithSudo("localhost", 2229, "root", "Password9", "/tmp/other-file.bin", "/etc/other-file.bin")
		require.NoError(t, err)
		fmt.Println()
	})

	t.Run("LongLivedConnection", func(t *testing.T) {
		c, err := Connect("localhost", 2221, "user2", "Password2")
		require.NoError(t, err)

		defer c.Close()

		defer func() {
			c.RunCommandWithSudo("rm -frv /var/log/weird-file.txt")
		}()

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
		require.NotEmpty(t, stderr)

		err = c.TransferFile("/tmp/some-file.txt", "/var/log/weird-file.txt")
		require.Error(t, err)

		err = c.TransferFileWithSudo("/tmp/some-file.txt", "/var/log/weird-file.txt")
		require.NoError(t, err)

		err = c.TransferFileWithSudo("/tmp/other-file.bin", "/var/log/weird-file.txt")
		require.NoError(t, err)

		stdout, stderr, err = c.RunCommandWithSudo("du -sh /var/log/weird-file.txt")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.NotEmpty(t, stderr)

		stdout, stderr, err = c.RunCommand("cd /srv/")
		require.NoError(t, err)

		stdout, stderr, err = c.RunCommand("pwd")
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
		require.Equal(t, stdout, "/home/user2\n")
	})
}
