package local

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Connection struct {
	Host              string
	Port              int
	Username          string
	Password          string
	Uname             string
	CanSudo           bool
	SudoNeedsPassword bool
	IsRoot            bool
	logger            *log.Logger
	tag               string
	stdoutPipe        io.ReadCloser
	stderrPipe        io.ReadCloser
	stdinPipe         io.WriteCloser
}

func Connect(host string, port int, username string, password string) (*Connection, error) {
	host = "localhost"
	port = 0

	if strings.HasPrefix(username, "$") {
		username = strings.Trim(username, "$")
		username = strings.Trim(username, "{")
		username = strings.Trim(username, "}")

		if username == "" {
			log.Printf("warning: empty username for Local::Connect; defaulting to ${USER}")
			username = "USER"
		}

		username = os.Getenv(username)
	}

	if strings.HasPrefix(password, "$") {
		password = strings.Trim(password, "$")
		password = strings.Trim(password, "{")
		password = strings.Trim(password, "}")

		if password == "" {
			log.Printf("warning: empty password for Local::Connect; defaulting to ${PASS}")
			password = "PASS"
		}

		passwordEnvVar := password

		password = os.Getenv(password)

		if password == "" {
			log.Printf("warning: empty password for Local::Connect after checking ${%s} env var; sudo will not be possible if it requires a password", passwordEnvVar)
		}
	}

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
		stdoutPipe: nil,
		stderrPipe: nil,
		stdinPipe:  nil,
	}

	c.logger.Printf("connected.")

	var err error

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

func (c *Connection) Close() {}

func (c *Connection) RunCommand(command string) (string, string, error) {
	cmd := exec.Command("bash", "-c", command)

	var err error

	c.stdoutPipe, err = cmd.StdoutPipe()
	if err != nil {
		return "", "", err
	}

	c.stderrPipe, err = cmd.StderrPipe()
	if err != nil {
		return "", "", err
	}

	c.stdinPipe, err = cmd.StdinPipe()
	if err != nil {
		return "", "", err
	}

	defer func() {
		_ = c.stdoutPipe.Close()
		_ = c.stderrPipe.Close()
		_ = c.stdinPipe.Close()
	}()

	wg := new(sync.WaitGroup)

	wg.Add(1)
	stdout := strings.Builder{}
	go func() {
		defer wg.Done()

		r := bufio.NewScanner(c.stdoutPipe)
		for {
			if !r.Scan() {
				break
			}

			line := r.Text()
			c.logger.Printf("LO <<< %s", line)
			stdout.WriteString(line)
			stdout.WriteRune('\n')
		}

		err := r.Err()
		if err != nil {
			c.logger.Printf("warning: stdout scanner experienced error %s", err)
		}

		b, _ := io.ReadAll(c.stdoutPipe)
		stdout.Write(b)
	}()

	wg.Add(1)
	stderr := strings.Builder{}
	go func() {
		defer wg.Done()

		r := bufio.NewScanner(c.stderrPipe)
		for {
			if !r.Scan() {
				break
			}

			line := r.Text()
			c.logger.Printf("LE <<< %s", line)
			stderr.WriteString(line)
			stderr.WriteRune('\n')
		}

		err := r.Err()
		if err != nil {
			c.logger.Printf("warning: stderr scanner experienced error %s", err)
		}

		b, _ := io.ReadAll(c.stderrPipe)
		stderr.Write(b)
	}()

	c.logger.Printf("LI >>> %s", command)
	err = cmd.Run()

	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	wg.Wait()

	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("failed to run command %#+v because %s; stderr: ...\n\n%s", command, err, stderr.String())
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
		return c.RunCommand(fmt.Sprintf("echo '%s' | sudo -S %s", c.Password, command))
	}

	return c.RunCommand(fmt.Sprintf("sudo -S %s", command))
}
func (c *Connection) Upload(localFilePath string, remoteFilePath string) error {
	// TODO(initialed): this is probably not the way
	localFilePath = strings.ReplaceAll(localFilePath, "${USER}", c.Username)
	remoteFilePath = strings.ReplaceAll(remoteFilePath, "${USER}", c.Username)

	c.logger.Printf("uploading %#+v to %#+v", localFilePath, remoteFilePath)

	_, _, err := c.RunCommand(fmt.Sprintf("cp -fr '%s' '%s'", localFilePath, remoteFilePath))
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s:%s because %s", localFilePath, c.tag, remoteFilePath, err)
	}

	_, _, _ = c.RunCommand(fmt.Sprintf("ls -al %#+v", remoteFilePath))

	c.logger.Printf("uploaded %#+v to %#+v", localFilePath, remoteFilePath)

	return nil
}

func (c *Connection) UploadWithSudo(localFilePath string, remoteFilePath string) error {
	// TODO(initialed): this is probably not the way
	localFilePath = strings.ReplaceAll(localFilePath, "${USER}", c.Username)
	remoteFilePath = strings.ReplaceAll(remoteFilePath, "${USER}", c.Username)

	if c.IsRoot {
		return c.Upload(localFilePath, remoteFilePath)
	}

	tempFile := fmt.Sprintf("/tmp/deployed-upload-file-%s.tmp", uuid.Must(uuid.NewRandom()))

	c.logger.Printf("uploading file %#+v to %#+v and then moving to %#+v with sudo", localFilePath, tempFile, remoteFilePath)

	if !c.CanSudo {
		return fmt.Errorf("this session cannot sudo (cannot move file after upload)")
	}

	err := c.Upload(localFilePath, tempFile)
	if err != nil {
		return err
	}

	_, _, err = c.RunCommandWithSudo(fmt.Sprintf("mv '%s' '%s'", tempFile, remoteFilePath))
	if err != nil {
		return err
	}

	_, _, _ = c.RunCommandWithSudo(fmt.Sprintf("ls -al %#+v", remoteFilePath))

	c.logger.Printf("uploaded %#+v to %#+v", localFilePath, remoteFilePath)

	return nil
}

func (c *Connection) Download(remoteFilePath string, localFilePath string) error {
	c.logger.Printf("downloading %#+v to %#+v", remoteFilePath, localFilePath)

	_, _, err := c.RunCommand(fmt.Sprintf("cp -fr '%s' '%s'", remoteFilePath, localFilePath))
	if err != nil {
		return fmt.Errorf("failed to download %s to %s:%s because %s", remoteFilePath, c.tag, localFilePath, err)
	}

	_, _, _ = c.RunCommand(fmt.Sprintf("ls -al %#+v", localFilePath))

	c.logger.Printf("downloaded %#+v to %#+v", remoteFilePath, localFilePath)

	return nil
}

func (c *Connection) DownloadWithSudo(remoteFilePath string, localFilePath string) error {
	if c.IsRoot {
		return c.Download(remoteFilePath, localFilePath)
	}

	if !c.CanSudo {
		return fmt.Errorf("this session cannot sudo (cannot copy file before transfer)")
	}

	tempFile := fmt.Sprintf("/tmp/deployed-download-file-%s.tmp", uuid.Must(uuid.NewRandom()))

	c.logger.Printf("copying file %#+v to %#+v with sudo and then downloading to %#+v", remoteFilePath, tempFile, localFilePath)

	_, _, err := c.RunCommandWithSudo(fmt.Sprintf("cp -fr '%s' '%s'", remoteFilePath, tempFile))
	if err != nil {
		return err
	}

	_, _, err = c.RunCommandWithSudo(fmt.Sprintf("chown '%s:%s' '%s'", c.Username, c.Username, tempFile))
	if err != nil {
		return err
	}

	_, _, err = c.RunCommandWithSudo(fmt.Sprintf("mv -f '%s' '%s'", tempFile, localFilePath))
	if err != nil {
		return err
	}

	return nil
}
