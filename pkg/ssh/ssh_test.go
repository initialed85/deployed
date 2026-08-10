package ssh

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

	t.Run("RunCommandOnHostWithNoSudoAtAll", func(t *testing.T) {
		c, err := Connect("localhost", 2221, "user1", "Password1")
		require.NoError(t, err)

		stdout, stderr, err := c.RunCommand("echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		require.Empty(t, stderr)
		require.Equal(t, "Hello, world.\n", stdout)
		fmt.Println()

		stdout, stderr, err = c.RunCommandWithSudo("echo 'Hello, world.'")
		require.Error(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		fmt.Println()

		err = c.UploadFile("/tmp/some-file.txt", "/home/user1/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFileWithSudo("/tmp/some-file.txt", "/etc/some-file.txt")
		require.Error(t, err)
		fmt.Println()

		err = c.UploadFileWithSudo("/tmp/other-file.bin", "/etc/other-file.bin")
		require.Error(t, err)
		fmt.Println()

		err = c.UploadFolder("/tmp/some-folder", "/home/user1/some-folder")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFolderWithSudo("/tmp/some-folder", "/etc/some-folder")
		require.Error(t, err)
		fmt.Println()
	})

	t.Run("RunCommandOnHostWithSudo", func(t *testing.T) {
		c, err := Connect("localhost", 2221, "user2", "Password2")
		require.NoError(t, err)

		stdout, stderr, err := c.RunCommand("echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		require.Empty(t, stderr)
		require.Equal(t, "Hello, world.\n", stdout)
		fmt.Println()

		stdout, stderr, err = c.RunCommandWithSudo("echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		fmt.Println()

		err = c.UploadFile("/tmp/some-file.txt", "/home/user2/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFileWithSudo("/tmp/some-file.txt", "/etc/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFileWithSudo("/tmp/other-file.bin", "/etc/other-file.bin")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFolder("/tmp/some-folder", "/home/user2/some-folder")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFolderWithSudo("/tmp/some-folder", "/etc/some-folder")
		require.NoError(t, err)
		fmt.Println()
	})

	t.Run("RunCommandOnHostWithSudoNoPassword", func(t *testing.T) {
		c, err := Connect("localhost", 2221, "user3", "Password3")
		require.NoError(t, err)

		stdout, stderr, err := c.RunCommand("echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		require.Empty(t, stderr)
		require.Equal(t, "Hello, world.\n", stdout)
		fmt.Println()

		stdout, stderr, err = c.RunCommandWithSudo("echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		fmt.Println()

		err = c.UploadFile("/tmp/some-file.txt", "/home/user3/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFileWithSudo("/tmp/some-file.txt", "/etc/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFileWithSudo("/tmp/other-file.bin", "/etc/other-file.bin")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFolder("/tmp/some-folder", "/home/user3/some-folder")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFolderWithSudo("/tmp/some-folder", "/etc/some-folder")
		require.NoError(t, err)
		fmt.Println()
	})

	t.Run("RunCommandSudoNotNeeded", func(t *testing.T) {
		c, err := Connect("localhost", 2229, "root", "Password9")
		require.NoError(t, err)

		stdout, stderr, err := c.RunCommand("echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		require.Empty(t, stderr)
		require.Equal(t, "Hello, world.\n", stdout)
		fmt.Println()

		stdout, stderr, err = c.RunCommandWithSudo("echo 'Hello, world.'")
		require.NoError(t, err, fmt.Sprintf("stdout: %#+v, stder: %#+v", stdout, stderr))
		fmt.Println()

		err = c.UploadFile("/tmp/some-file.txt", "/root/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFileWithSudo("/tmp/some-file.txt", "/etc/some-file.txt")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFileWithSudo("/tmp/other-file.bin", "/etc/other-file.bin")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFolder("/tmp/some-folder", "/root/some-folder")
		require.NoError(t, err)
		fmt.Println()

		err = c.UploadFolderWithSudo("/tmp/some-folder", "/etc/some-folder")
		require.NoError(t, err)
		fmt.Println()
	})

	t.Run("LongLivedConnection", func(t *testing.T) {
		c, err := Connect("localhost", 2221, "user2", "Password2")
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
		require.NotEmpty(t, stderr)

		err = c.Upload("/tmp/some-file.txt", "/var/log/weird-file.txt")
		require.Error(t, err)

		err = c.UploadWithSudo("/tmp/some-file.txt", "/var/log/weird-file.txt")
		require.NoError(t, err)

		err = c.UploadWithSudo("/tmp/other-file.bin", "/var/log/weird-file.txt")
		require.NoError(t, err)

		err = c.Upload("/tmp/some-folder", "/home/user2/some-folder")
		require.NoError(t, err)

		err = c.UploadWithSudo("/tmp/some-folder", "/var/log/weird-folder")
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

		err = c.DownloadWithSudo("/etc", "/tmp/etc")
		require.NoError(t, err)

		b, err = os.ReadFile("/tmp/etc/hostname")
		require.NoError(t, err)
		require.NotEmpty(t, b)
	})
}
