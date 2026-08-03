package ssh

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

type Connection struct {
	Host              string
	Port              int
	Username          string
	Password          string
	SSHConfig         *ssh.ClientConfig
	SSHClient         *ssh.Client
	SCPClient         *scp.Client
	SSHSession        *ssh.Session
	Uname             string
	CanSudo           bool
	SudoNeedsPassword bool
	IsRoot            bool
	logger            *log.Logger
	tag               string
	clientMu          *sync.Mutex
	sessionMu         *sync.Mutex
	stdout            io.Reader
	stderr            io.Reader
	stdin             io.Writer
}

func Connect(host string, port int, username string, password string) (*Connection, error) {
	tag := fmt.Sprintf("%s@%s:%d", username, host, port)

	c := &Connection{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		tag:      tag,
		logger: log.New(
			os.Stdout,
			fmt.Sprintf("Connection{%s} ", tag),
			log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC|log.Lmsgprefix,
		),
		clientMu:  new(sync.Mutex),
		sessionMu: new(sync.Mutex),
		stdout:    bytes.NewBuffer(nil),
		stderr:    bytes.NewBuffer(nil),
		stdin:     bytes.NewBuffer(nil),
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

	var err error

	c.logger.Printf("connecting to %s...", c.tag)

	c.SSHClient, err = ssh.Dial(
		"tcp",
		fmt.Sprintf("%s:%d", host, port),
		c.SSHConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s because %s", c.tag, err)
	}

	scpClient, err := scp.NewClientBySSH(c.SSHClient)
	if err != nil {
		return nil, fmt.Errorf("failed to add SCP client to %s because %s", c.tag, err)
	}

	c.SCPClient = &scpClient

	c.logger.Printf("connected.")

	c.Uname, _, err = c.RunCommand("uname -a")
	if err != nil {
		c.Close()
		return nil, err
	}
	c.Uname = strings.TrimSpace(c.Uname)

	c.IsRoot = c.Username == "root"

	if !c.IsRoot {
		_, _, err := c.RunCommand("sudo -n true")
		if err != nil {
			_, _, err = c.RunCommand(fmt.Sprintf("echo '%s' | sudo -S true", c.Password))
			c.CanSudo = err == nil
			c.SudoNeedsPassword = c.CanSudo
		} else {
			c.CanSudo = true
			c.SudoNeedsPassword = false
		}
	}

	c.logger.Printf("is root: %v", c.IsRoot)
	if !c.IsRoot {
		c.logger.Printf("can sudo: %v", c.CanSudo)
		if c.CanSudo {
			c.logger.Printf("sudo needs password: %v", c.SudoNeedsPassword)
		}
	}

	return c, nil
}

func (c *Connection) Close() {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()

	c.logger.Printf("closing...")

	if c.SSHClient == nil {
		c.logger.Printf("warning: already closed.")
		return
	}

	if c.SSHSession != nil {
		_ = c.SSHSession.Close()
	}

	if c.SSHClient != nil {
		_ = c.SSHClient.Close()
	}

	c.logger.Printf("closed.")
}

func (c *Connection) RunCommand(command string) (string, string, error) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()

	var err error

	c.SSHSession, err = c.SSHClient.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to get session for %s because %s", c.tag, err)
	}
	defer c.SSHSession.Close()

	c.stdout, err = c.SSHSession.StdoutPipe()
	if err != nil {
		return "", "", err
	}

	c.stderr, err = c.SSHSession.StderrPipe()
	if err != nil {
		return "", "", err
	}

	c.stdin, err = c.SSHSession.StdinPipe()
	if err != nil {
		return "", "", err
	}

	wg := new(sync.WaitGroup)

	wg.Add(1)
	stdout := strings.Builder{}
	go func() {
		defer wg.Done()

		r := bufio.NewScanner(c.stdout)
		for {
			if !r.Scan() {
				break
			}

			line := r.Text()
			c.logger.Printf("O <<< %s", line)
			stdout.WriteString(line)
			stdout.WriteRune('\n')
		}

		err := r.Err()
		if err != nil {
			c.logger.Printf("warning: stdout scanner experienced error %s", err)
		}

		b, _ := io.ReadAll(c.stdout)
		stdout.Write(b)
	}()

	wg.Add(1)
	stderr := strings.Builder{}
	go func() {
		defer wg.Done()

		r := bufio.NewScanner(c.stderr)
		for {
			if !r.Scan() {
				break
			}

			line := r.Text()
			c.logger.Printf("E <<< %s", line)
			stderr.WriteString(line)
			stderr.WriteRune('\n')
		}

		err := r.Err()
		if err != nil {
			c.logger.Printf("warning: stderr scanner experienced error %s", err)
		}

		b, _ := io.ReadAll(c.stderr)
		stderr.Write(b)
	}()

	c.logger.Printf("I >>> %s", command)
	err = c.SSHSession.Run(command)
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("failed to run command %#+v because %s; stderr: ...\n\n%s", command, err, stderr.String())
	}
	_ = c.SSHSession.Close()

	wg.Wait()

	return stdout.String(), stderr.String(), nil
}

func (c *Connection) RunCommandWithSudo(command string) (string, string, error) {
	if c.IsRoot {
		return c.RunCommand(command)
	}

	if !c.CanSudo {
		return "", "", fmt.Errorf("this session cannot sudo")
	}

	if c.SudoNeedsPassword {
		return c.RunCommand(fmt.Sprintf("echo '%s' | sudo -S %s", c.Password, command))
	}

	return c.RunCommand(fmt.Sprintf("sudo -S %s", command))
}

func (c *Connection) TransferFile(localFilePath string, remoteFilePath string) error {
	stat, err := os.Stat(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to transfer %s to %s:%s because %s", localFilePath, c.tag, remoteFilePath, err)
	}

	f, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to transfer %s to %s:%s because %s", localFilePath, c.tag, remoteFilePath, err)
	}
	defer f.Close()

	err = c.SCPClient.CopyFromFile(
		context.Background(),
		*f,
		remoteFilePath,
		"0"+strconv.FormatInt(int64(stat.Mode().Perm()), 8),
	)
	if err != nil {
		return fmt.Errorf("failed to transfer %s to %s:%s because %s", localFilePath, c.tag, remoteFilePath, err)
	}

	return nil
}

func (c *Connection) TransferFileWithSudo(localFilePath string, remoteFilePath string) error {
	if c.IsRoot {
		return c.TransferFile(localFilePath, remoteFilePath)
	}

	if !c.CanSudo {
		return fmt.Errorf("this session cannot sudo (cannot move file after transfer)")
	}

	tempFile := fmt.Sprintf("/tmp/deployed-transfer-%s.tmp", uuid.Must(uuid.NewRandom()))

	err := c.TransferFile(localFilePath, tempFile)
	if err != nil {
		return err
	}

	_, _, err = c.RunCommandWithSudo(fmt.Sprintf("mv '%s' '%s'", tempFile, remoteFilePath))
	if err != nil {
		return err
	}

	return nil
}

func RunCommand(host string, port int, username string, password string, command string) (string, string, error) {
	c, err := Connect(host, port, username, password)
	if err != nil {
		return "", "", err
	}
	defer c.Close()

	return c.RunCommand(command)
}

func RunCommandWithSudo(host string, port int, username string, password string, command string) (string, string, error) {
	c, err := Connect(host, port, username, password)
	if err != nil {
		return "", "", err
	}
	defer c.Close()

	return c.RunCommandWithSudo(command)
}

func TransferFile(host string, port int, username string, password string, localFilePath string, remoteFilePath string) error {
	c, err := Connect(host, port, username, password)
	if err != nil {
		return err
	}
	defer c.Close()

	return c.TransferFile(localFilePath, remoteFilePath)
}

func TransferFileWithSudo(host string, port int, username string, password string, localFilePath string, remoteFilePath string) error {
	c, err := Connect(host, port, username, password)
	if err != nil {
		return err
	}
	defer c.Close()

	return c.TransferFileWithSudo(localFilePath, remoteFilePath)
}
