package local

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/initialed85/deployed/pkg/connection/connection_types"
)

const LocalConnectionHostSentinel = "__local__"
const LocalConnectionPortSentinel = int(0)

type Connection struct {
	logger   *log.Logger
	Host     string
	Port     int
	Username string
	Password string
}

func Open(host string, port int, username string, password string) (connection_types.Connectable, error) {
	if host != LocalConnectionHostSentinel {
		return nil, fmt.Errorf("assertion failed: local connection must have host set to %#+v", LocalConnectionHostSentinel)
	}

	if port > LocalConnectionPortSentinel {
		return nil, fmt.Errorf("assertion failed: local connection must have port set to 0")
	}

	c := &Connection{
		logger: log.New(
			os.Stdout,
			fmt.Sprintf("Local{%s@%s:%d} ", username, host, port),
			log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC|log.Lmsgprefix,
		),
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}

	confirmedUsername := strings.TrimSpace(os.Getenv("USER"))

	if confirmedUsername == "" {
		rawUsernameFromWhoami, _ := exec.Command("bash", "-c", "whoami").Output()
		confirmedUsername = strings.TrimSpace(string(rawUsernameFromWhoami))
	}

	if confirmedUsername != "" {
		c.Username = confirmedUsername
	}

	return c, nil
}

func (c *Connection) GetUsername() string {
	return c.Username
}

func (c *Connection) SetLocalConnection(localConnection connection_types.Deployable) {
	// noop
}

func (c *Connection) Close() {
	// noop
}

// homeDir is the working directory a local command starts in.
//
// It used to be built from the username with a darwin/linux split (/home/<user> or /Users/<user>).
// That is wrong for exactly the cases that reach CI: a local connection never switches user, so in a
// container running as root it produced /home/root, which does not exist, and every command then
// failed with a chdir error instead of the error the caller was testing for. The same guess breaks for
// any home that is not under the conventional prefix. The process's own identity is the only relevant
// one here, so the OS is the authority on its home; the path construction survives only as the
// fallback when HOME is unset.
func (c *Connection) homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	if runtime.GOOS != "darwin" {
		return fmt.Sprintf("/home/%s", c.Username)
	}
	return fmt.Sprintf("/Users/%s", c.Username)
}

func (c *Connection) PrepareCommand(command string) (io.Reader, io.Reader, io.Writer, connection_types.RunPreparedCommandFn, context.CancelFunc, error) {
	cmd := exec.Command("sh", "-c", command)

	cmd.Dir = c.homeDir()

	cmd.Env = make([]string, 0)

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "PWD=") {
			cmd.Env = append(cmd.Env, fmt.Sprintf("PWD=%s", cmd.Dir))
		} else if env == "USER=" {
			continue
		} else {
			cmd.Env = append(cmd.Env, env)
		}
	}

	// A spec writes ${USER} in upload and download paths, where this connection resolves it to the
	// username it confirmed above.  A script is only a shell, so it has to be told the same thing or the
	// two halves of one step disagree: in a container (which ships no USER) the paths said /home/root/...
	// while the script said /home//....
	if os.Getenv("USER") == "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("USER=%s", c.Username))
	}

	// exec's own pipes are closed by cmd.Run, which can happen while the caller is still reading them,
	// and everything left in the buffer is lost.  These write ends are ours, so they close after the
	// command has finished and the readers see every byte, then EOF.
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-ctx.Done()

		_ = stdoutWriter.Close()
		_ = stderrWriter.Close()
		_ = stdinPipe.Close()

	}()

	run := func() error {
		err := cmd.Run()

		_ = stdoutWriter.Close()
		_ = stderrWriter.Close()

		return err
	}

	return stdoutReader, stderrReader, stdinPipe, run, cancel, nil
}

func (c *Connection) doCopy(runCommandFn connection_types.RunCommandFn, localPath string, remotePath string) error {
	localPath = strings.ReplaceAll(localPath, "${USER}", c.Username)
	remotePath = strings.ReplaceAll(remotePath, "${USER}", c.Username)

	// step scripts are uploaded to the same path they occupy locally, so a local connection
	// would be asked to copy a file onto itself, which rsync rejects
	if localPath == remotePath {
		return nil
	}

	stat, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		if !strings.HasSuffix(localPath, "/") {
			localPath += "/"
		}

		if !strings.HasSuffix(remotePath, "/") {
			remotePath += "/"
		}

		_ = os.MkdirAll(remotePath, 0o777)
	} else {
		_ = os.MkdirAll(filepath.Dir(remotePath), 0o777)
	}

	_, _, err = runCommandFn(fmt.Sprintf("rsync -avc '%s' '%s'", localPath, remotePath))
	if err != nil {
		return err
	}

	return nil
}

func (c *Connection) UploadFile(runCommandFn connection_types.RunCommandFn, needSudo bool, localPath string, remotePath string) error {
	return c.doCopy(runCommandFn, localPath, remotePath)
}

func (c *Connection) UploadFolder(runCommandFn connection_types.RunCommandFn, needSudo bool, localPath string, remotePath string) error {
	return c.doCopy(runCommandFn, localPath, remotePath)
}

func (c *Connection) DownloadFile(runCommandFn connection_types.RunCommandFn, needSudo bool, remotePath string, localPath string) error {
	return c.doCopy(runCommandFn, remotePath, localPath)
}

func (c *Connection) DownloadFolder(runCommandFn connection_types.RunCommandFn, needSudo bool, remotePath string, localPath string) error {
	return c.doCopy(runCommandFn, remotePath, localPath)
}
