package ssh

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/connection/connection_types"
	"github.com/initialed85/deployed/pkg/connection/local"
	"github.com/initialed85/deployed/pkg/helpers/env"
	"golang.org/x/crypto/ssh"
)

type Connection struct {
	logger          *log.Logger
	Host            string
	Port            int
	Username        string
	Password        string
	SSHConfig       *ssh.ClientConfig
	SSHClient       *ssh.Client
	SCPClient       *scp.Client
	mu              *sync.Mutex
	localConnection connection_types.Deployable
}

func Open(host string, port int, username string, password string) (connection_types.Connectable, error) {
	if host == local.LocalConnectionHostSentinel {
		return nil, fmt.Errorf("assertion failed: local connection must not have host set to %#+v", local.LocalConnectionHostSentinel)
	}

	if port <= local.LocalConnectionPortSentinel {
		return nil, fmt.Errorf("assertion failed: local connection must have port > 0")
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
		mu:       new(sync.Mutex),
	}

	c.SSHConfig = &ssh.ClientConfig{}

	c.SSHConfig.SetDefaults()

	c.SSHConfig.Ciphers = append(
		c.SSHConfig.Ciphers,
		"aes128-cbc",
		"3des-cbc",
		"aes256-cbc",
		"twofish256-cbc",
		"twofish-cbc",
		"twofish128-cbc",
		"blowfish-cbc",
	)

	c.SSHConfig.User = username
	c.SSHConfig.Auth = []ssh.AuthMethod{ssh.Password(password)}
	c.SSHConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey() // TODO(initialed85): better security
	c.SSHConfig.Timeout = time.Second * 10

	if env.IsDebug {
		c.logger.Printf("connecting to %s...", strings.TrimSpace(c.logger.Prefix()))
	}

	var err error

	c.SSHClient, err = ssh.Dial(
		"tcp",
		fmt.Sprintf("%s:%d", host, port),
		c.SSHConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s:%d because %s", c.Host, c.Port, err)
	}

	scpClient, err := scp.NewClientBySSH(c.SSHClient)
	if err != nil {
		return nil, fmt.Errorf("failed to add SCP client to %s@%s:%d because %s", c.Username, c.Host, c.Port, err)
	}

	c.SCPClient = &scpClient

	if env.IsDebug {
		c.logger.Printf("connected.")
	}

	sshSession, err := c.SSHClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH session for %s@%s:%d because %s", c.Username, c.Host, c.Port, err)
	}
	defer sshSession.Close()

	stdout, _ := sshSession.Output("echo ${USER}")

	confirmedUsername := strings.TrimSpace(string(stdout))

	if confirmedUsername != "" {
		c.Username = confirmedUsername
	}

	return c, nil
}

func (c *Connection) GetUsername() string {
	return c.Username
}

func (c *Connection) SetLocalConnection(localConnection connection_types.Deployable) {
	c.localConnection = localConnection
}

func (c *Connection) Close() {
	if c.SSHClient != nil {
		if c.SSHClient.Conn != nil {
			_ = c.SSHClient.Conn.Close()
		}

		_ = c.SSHClient.Close()
	}

	if c.SCPClient != nil {
		c.SCPClient.Close()
	}
}

func (c *Connection) PrepareCommand(command string) (io.Reader, io.Reader, io.Writer, connection_types.RunPreparedCommandFn, context.CancelFunc, error) {
	//
	// TODO: all the SSH shit in here- pass cleanup out through the cancel func
	//

	sshSession, err := c.SSHClient.NewSession()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to get SSH session for %s:%d because %s", c.Host, c.Port, err)
	}

	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to get SSH session for %s:%d because %s", c.Host, c.Port, err)
	}

	stderrPipe, err := sshSession.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to get SSH session for %s:%d because %s", c.Host, c.Port, err)
	}

	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to get SSH session for %s:%d because %s", c.Host, c.Port, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-ctx.Done()

		_ = sshSession.Close()
		_ = stdinPipe.Close()
	}()

	runFn := func() error {
		return sshSession.Run(command)
	}

	return stdoutPipe, stderrPipe, stdinPipe, runFn, cancel, nil
}

func (c *Connection) UploadFile(runCommandFn connection_types.RunCommandFn, needSudo bool, localPath string, remotePath string) error {
	localPath = strings.ReplaceAll(localPath, "${USER}", c.localConnection.GetUsername())
	remotePath = strings.ReplaceAll(remotePath, "${USER}", c.GetUsername())

	stat, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePath, err)
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePath, err)
	}
	defer f.Close()

	var remotePathForTransfer string

	if needSudo {
		remotePathForTransfer = fmt.Sprintf("/tmp/deployed-upload-file-%s.tmp", uuid.Must(uuid.NewRandom()))
	} else {
		remotePathForTransfer = remotePath
	}

	if env.IsDebug {
		c.logger.Printf("uploading %s to %s via SCP", localPath, remotePathForTransfer)
	}

	if !needSudo {
		remoteFolderPath, _ := filepath.Split(remotePath)

		if len(remoteFolderPath) > 0 {
			_, _, err = runCommandFn(fmt.Sprintf("mkdir -p '%s'", remoteFolderPath))
			if err != nil {
				return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePath, err)
			}
		}
	}

	err = c.SCPClient.CopyFromFile(
		context.Background(),
		*f,
		remotePathForTransfer,
		"0"+strconv.FormatInt(int64(stat.Mode().Perm()), 8),
	)
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePathForTransfer, err)
	}

	if needSudo {
		remoteFolderPath, _ := filepath.Split(remotePath)

		if len(remoteFolderPath) > 0 {
			_, _, err = runCommandFn(fmt.Sprintf("mkdir -p '%s'", remoteFolderPath))
			if err != nil {
				return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePath, err)
			}
		}

		_, _, err = runCommandFn(fmt.Sprintf("mv -fv '%s' '%s'", remotePathForTransfer, remotePath))
		if err != nil {
			return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePath, err)
		}
	}

	return nil
}

func (c *Connection) UploadFolder(runCommandFn connection_types.RunCommandFn, needSudo bool, localPath string, remotePath string) error {
	localPath = strings.ReplaceAll(localPath, "${USER}", c.localConnection.GetUsername())
	remotePath = strings.ReplaceAll(remotePath, "${USER}", c.GetUsername())

	tempFile := fmt.Sprintf("/tmp/deployed-upload-folder-%s.tar.gz", uuid.Must(uuid.NewRandom()))

	_, _, err := c.localConnection.RunCommand(fmt.Sprintf("tar czvf '%s' -C '%s' .", tempFile, localPath))
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePath, err)
	}

	err = c.UploadFile(runCommandFn, needSudo, tempFile, tempFile)
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePath, err)
	}

	_, _, err = runCommandFn(fmt.Sprintf("mkdir -p '%s'", remotePath))
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePath, err)
	}

	_, _, err = runCommandFn(fmt.Sprintf("tar xzvf '%s' -C '%s'", tempFile, remotePath))
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s@%s:%d:%s because %s", localPath, c.Username, c.Host, c.Port, remotePath, err)
	}

	return nil
}

func (c *Connection) DownloadFile(runCommandFn connection_types.RunCommandFn, needSudo bool, remotePath string, localPath string) error {
	localPath = strings.ReplaceAll(localPath, "${USER}", c.localConnection.GetUsername())
	remotePath = strings.ReplaceAll(remotePath, "${USER}", c.GetUsername())

	var remotePathForTransfer string
	var localPathForTransfer string

	if needSudo {
		remotePathForTransfer = fmt.Sprintf("/tmp/deployed-download-file-%s.tmp", uuid.Must(uuid.NewRandom()))
		localPathForTransfer = remotePathForTransfer
	} else {
		remotePathForTransfer = remotePath
		localPathForTransfer = localPath
	}

	f, err := os.Create(localPathForTransfer)
	if err != nil {
		return fmt.Errorf("failed to download file %s to %s@%s:%d:%s because %s", remotePath, c.Username, c.Host, c.Port, localPath, err)
	}

	defer func() {
		_ = f.Close()
	}()

	if env.IsDebug {
		c.logger.Printf("downloading %s to %s via SCP", remotePathForTransfer, localPath)
	}

	if needSudo {
		_, _, err := runCommandFn(fmt.Sprintf("cp -fv '%s' '%s'", remotePath, remotePathForTransfer))
		if err != nil {
			return fmt.Errorf("failed to download file %s to %s@%s:%d:%s because %s", remotePath, c.Username, c.Host, c.Port, localPath, err)
		}

		_, _, err = runCommandFn(fmt.Sprintf("chown -fRv '%s:%s' '%s'", c.Username, c.Username, remotePathForTransfer))
		if err != nil {
			return fmt.Errorf("failed to download file %s to %s@%s:%d:%s because %s", remotePath, c.Username, c.Host, c.Port, localPath, err)
		}
	}

	err = c.SCPClient.CopyFromRemote(
		context.Background(),
		f,
		remotePathForTransfer,
	)
	if err != nil {
		return fmt.Errorf("failed to download file %s to %s@%s:%d:%s because %s", remotePathForTransfer, c.Username, c.Host, c.Port, localPath, err)
	}

	err = f.Sync()
	if err != nil {
		return fmt.Errorf("failed to download file %s to %s@%s:%d:%s because %s", remotePathForTransfer, c.Username, c.Host, c.Port, localPath, err)
	}

	_ = f.Close()

	if needSudo {
		_, _, err := c.localConnection.RunCommandWithSudo(fmt.Sprintf("mv -fv '%s' '%s'", localPathForTransfer, localPath))
		if err != nil {
			return fmt.Errorf("failed to download file %s to %s@%s:%d:%s because %s", remotePath, c.Username, c.Host, c.Port, localPath, err)
		}
	}

	return nil
}

func (c *Connection) DownloadFolder(runCommandFn connection_types.RunCommandFn, needSudo bool, remotePath string, localPath string) error {
	localPath = strings.ReplaceAll(localPath, "${USER}", c.localConnection.GetUsername())
	remotePath = strings.ReplaceAll(remotePath, "${USER}", c.GetUsername())

	tempFile := fmt.Sprintf("/tmp/deployed-download-folder-%s.tar.gz", uuid.Must(uuid.NewRandom()))

	_, _, err := runCommandFn(fmt.Sprintf("tar czf '%s' -C '%s' .", tempFile, remotePath))
	if err != nil {
		return fmt.Errorf("failed to download folder %s to %s@%s:%d:%s because %s", remotePath, c.Username, c.Host, c.Port, localPath, err)
	}

	err = c.DownloadFile(runCommandFn, needSudo, tempFile, tempFile)
	if err != nil {
		return fmt.Errorf("failed to download folder %s to %s@%s:%d:%s because %s", tempFile, c.Username, c.Host, c.Port, tempFile, err)
	}

	if needSudo {
		_, _, err = c.localConnection.RunCommandWithSudo(fmt.Sprintf("mkdir -p '%s'", localPath))
	} else {
		_, _, err = c.localConnection.RunCommand(fmt.Sprintf("mkdir -p '%s'", localPath))
	}

	if err != nil {
		return fmt.Errorf("failed to download folder %s to %s@%s:%d:%s because %s", tempFile, c.Username, c.Host, c.Port, tempFile, err)
	}

	if needSudo {
		_, _, err = c.localConnection.RunCommandWithSudo(fmt.Sprintf("tar xzf '%s' -C '%s'", tempFile, localPath))
	} else {
		_, _, err = c.localConnection.RunCommand(fmt.Sprintf("tar xzf '%s' -C '%s'", tempFile, localPath))
	}

	if err != nil {
		return fmt.Errorf("failed to download folder %s to %s@%s:%d:%s because %s", tempFile, c.Username, c.Host, c.Port, tempFile, err)
	}

	return nil
}
