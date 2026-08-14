package connection

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/initialed85/deployed/pkg/connection/connection_types"
	"github.com/initialed85/deployed/pkg/helpers/env"
)

type Connection struct {
	logger            *log.Logger
	c                 connection_types.Connectable
	Host              string
	Port              int
	Username          string
	Password          string
	Uname             string
	CanSudo           bool
	SudoNeedsPassword bool
	IsRoot            bool
}

func open(host string, port int, username string, password string, openConnectableFn connection_types.OpenConnectableFn, connectableType string) (connection_types.Deployable, error) {
	c := &Connection{
		logger: log.New(
			os.Stdout,
			fmt.Sprintf("Conn[%s]{%s@%s:%d} ", connectableType, username, host, port),
			log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC|log.Lmsgprefix,
		),
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}

	var err error

	c.c, err = openConnectableFn(host, port, username, password)
	if err != nil {
		return nil, err
	}

	c.logger.SetPrefix(fmt.Sprintf("Conn[%s]{%s@%s:%d} ", connectableType, c.c.GetUsername(), host, port))

	c.Uname, _, err = c.RunCommand("uname -a")
	if err != nil {
		c.Close()
		return nil, err
	}
	c.Uname = strings.TrimSpace(c.Uname)

	c.IsRoot = c.Username == "root"

	if !c.IsRoot {
		_, _, err := c.RunCommand("sudo -s -S -n true")
		if err != nil {
			_, _, err = c.RunCommand(fmt.Sprintf("echo '%s' | sudo -s -S true", c.Password))
			c.CanSudo = err == nil
			c.SudoNeedsPassword = c.CanSudo
		} else {
			c.CanSudo = true
			c.SudoNeedsPassword = false
		}
	}

	if env.IsDebug {
		c.logger.Printf("is root: %v", c.IsRoot)
		if !c.IsRoot {
			c.logger.Printf("can sudo: %v", c.CanSudo)
			if c.CanSudo {
				c.logger.Printf("sudo needs password: %v", c.SudoNeedsPassword)
			}
		}
	}

	return c, nil
}

func (c *Connection) GetUsername() string {
	return c.c.GetUsername()
}

func (c *Connection) GetConnectable() connection_types.Connectable {
	return c.c
}

func (c *Connection) Close() {
	c.c.Close()
}

func (c *Connection) RunCommand(command string) (string, string, error) {
	stdoutPipe, stderrPipe, stdinPipe, runFn, cancelFn, err := c.c.PrepareCommand(command)
	if err != nil {
		return "", "", err
	}

	_ = stdinPipe

	defer cancelFn()

	wg := new(sync.WaitGroup)

	wg.Add(1)
	stdout := strings.Builder{}
	go func() {
		defer wg.Done()

		r := bufio.NewScanner(stdoutPipe)
		for {
			if !r.Scan() {
				break
			}

			line := r.Text()
			if env.IsDebug {
				c.logger.Printf("LO <<< %s", line)
			}
			stdout.WriteString(line)
			stdout.WriteRune('\n')
		}

		err := r.Err()
		if err != nil {
			c.logger.Printf("warning: stdout scanner experienced error %s", err)
		}

		b, _ := io.ReadAll(stdoutPipe)
		stdout.Write(b)
	}()

	wg.Add(1)
	stderr := strings.Builder{}
	go func() {
		defer wg.Done()

		r := bufio.NewScanner(stderrPipe)
		for {
			if !r.Scan() {
				break
			}

			line := r.Text()
			if env.IsDebug {
				c.logger.Printf("LE <<< %s", line)
			}
			stderr.WriteString(line)
			stderr.WriteRune('\n')
		}

		err := r.Err()
		if err != nil {
			c.logger.Printf("warning: stderr scanner experienced error %s", err)
		}

		b, _ := io.ReadAll(stderrPipe)
		stderr.Write(b)
	}()

	if env.IsDebug {
		c.logger.Printf("LI >>> %s", command)
	}
	err = runFn()

	cancelFn()

	wg.Wait()

	// the error might have been instant, but we wait to ensure we can fully drain stdout and stderr
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("failed to run command %#+v because %s; stderr: ...\n\n%s", command, err, stderr.String())
	}

	if stderr.Len() > 0 {
		stderrData := stderr.String()

		if strings.HasPrefix(stderrData, "[sudo:") && strings.Count(stderrData, "\n") <= 1 {
			stderr.Reset()
		}
	}

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
		return c.RunCommand(fmt.Sprintf("echo '%s' | sudo -s -S %s", c.Password, command))
	}

	return c.RunCommand(fmt.Sprintf("sudo -s -S %s", command))
}

func (c *Connection) Upload(localPath string, remotePath string) error {
	stat, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("upload of %#+v to %#+v failed because %s", localPath, remotePath, err)
	}

	if env.IsDebug {
		c.logger.Printf("uploading %#+v to %#+v", localPath, remotePath)
	}

	if !stat.IsDir() {
		err = c.c.UploadFile(c.RunCommand, false, localPath, remotePath)
	} else {
		err = c.c.UploadFolder(c.RunCommand, false, localPath, remotePath)
	}

	if err != nil {
		return err
	}

	return nil
}

func (c *Connection) UploadWithSudo(localPath string, remotePath string) error {
	stat, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("upload of %#+v to %#+v with sudo failed because %s", localPath, remotePath, err)
	}

	if env.IsDebug {
		c.logger.Printf("uploading %#+v to %#+v with sudo", localPath, remotePath)
	}

	if !stat.IsDir() {
		err = c.c.UploadFile(c.RunCommandWithSudo, true, localPath, remotePath)
	} else {
		err = c.c.UploadFolder(c.RunCommandWithSudo, true, localPath, remotePath)
	}

	if err != nil {
		return err
	}

	return nil
}

func (c *Connection) Download(remotePath string, localPath string) error {
	_, _, err := c.RunCommand(fmt.Sprintf("test -d '%s'", remotePath))
	isDir := err == nil

	if env.IsDebug {
		c.logger.Printf("downloading %#+v to %#+v", remotePath, localPath)
	}

	if !isDir {
		err = c.c.DownloadFile(c.RunCommand, false, remotePath, localPath)
	} else {
		err = c.c.DownloadFolder(c.RunCommand, false, remotePath, localPath)
	}

	if err != nil {
		return err
	}

	return nil
}

func (c *Connection) DownloadWithSudo(remotePath string, localPath string) error {
	_, _, err := c.RunCommandWithSudo(fmt.Sprintf("test -d '%s'", remotePath))
	isDir := err == nil

	if env.IsDebug {
		c.logger.Printf("downloading %#+v to %#+v with sudo", remotePath, localPath)
	}

	if !isDir {
		err = c.c.DownloadFile(c.RunCommandWithSudo, true, remotePath, localPath)
	} else {
		err = c.c.DownloadFolder(c.RunCommandWithSudo, true, remotePath, localPath)
	}

	if err != nil {
		return err
	}

	return nil
}
