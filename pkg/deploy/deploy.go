package deploy

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/ssh"
	"github.com/initialed85/deployed/pkg/types"
)

func Deploy(target string, steps []types.Step) error {
	stepsHash, err := types.HashSteps(steps)
	if err != nil {
		return err
	}

	deployID := uuid.Must(uuid.NewRandom())

	localStepsHashPath := fmt.Sprintf("/tmp/deployed-%s-steps-hash.txt", deployID)

	err = os.WriteFile(localStepsHashPath, []byte(stepsHash), 0o777)
	if err != nil {
		return err
	}

	defer func() {
		_ = os.Remove(localStepsHashPath)
	}()

	var username, password, host string
	var port int64

	err = func() error {
		parts := strings.Split(target, "@")
		if len(parts) < 2 {
			return fmt.Errorf("target doesn't have '@' symbol")
		}

		usernameAndPassword := parts[0]
		usernameAndPasswordParts := strings.Split(usernameAndPassword, ":")
		if len(usernameAndPasswordParts) < 2 {
			return fmt.Errorf("username-and-password (left) half doesn't have ':' symbol")
		}
		username = usernameAndPasswordParts[0]
		password = usernameAndPasswordParts[1]

		hostAndPort := parts[1]
		hostAndPortParts := strings.Split(hostAndPort, ":")
		if len(hostAndPortParts) < 2 {
			return fmt.Errorf("host-and-port (right) half doesn't have ':' symbol")
		}
		host = hostAndPortParts[0]
		rawPort := hostAndPortParts[1]

		var err error
		port, err = strconv.ParseInt(rawPort, 10, 32)
		if err != nil {
			return fmt.Errorf("port could not be parsed as int32 because %s", err)
		}

		return nil
	}()
	if err != nil {
		return fmt.Errorf("failed to split %#+v into (username:password)@(host:port) because %s", target, err)
	}

	c, err := ssh.Connect(host, int(port), username, password)
	if err != nil {
		return err
	}

	remoteAttemptedStepsHashPath := fmt.Sprintf("deployed-steps-hash.txt.attempted-%s", deployID)

	err = c.TransferFile(localStepsHashPath, remoteAttemptedStepsHashPath)
	if err != nil {
		return err
	}

	pathsToCleanUp := make([]string, 0)

	for i, step := range steps {
		err := func() error {
			if step.Script != "" {
				path := fmt.Sprintf("/tmp/deployed-%s-step-%d-script.sh", deployID, i)

				err = os.WriteFile(path, []byte(step.Script), 0o777)
				if err != nil {
					return err
				}

				err = c.TransferFile(path, path)
				if err != nil {
					return err
				}

				pathsToCleanUp = append(pathsToCleanUp, path)

				_, _, err := c.RunCommand(path)
				if err != nil {
					return err
				}
			}

			return nil
		}()
		if err != nil {
			return fmt.Errorf("failed to execute step %d script because %s", i, err)
		}

		err = func() error {
			if step.ScriptWithSudo != "" {
				path := fmt.Sprintf("/tmp/deployed-%s-step-%d-script.sh", deployID, i)

				err = os.WriteFile(path, []byte(step.ScriptWithSudo), 0o777)
				if err != nil {
					return err
				}

				err = c.TransferFile(path, path)
				if err != nil {
					return err
				}

				pathsToCleanUp = append(pathsToCleanUp, path)

				_, _, err := c.RunCommandWithSudo(path)
				if err != nil {
					return err
				}
			}

			return nil
		}()
		if err != nil {
			return fmt.Errorf("failed to execute step %d script_with_sudo because %s", i, err)
		}
	}

	for _, path := range pathsToCleanUp {
		_, _, _ = c.RunCommand(fmt.Sprintf("rm -f %s", path))
	}

	remoteStepsHashPath := "deployed-steps-hash.txt"

	_, _, _ = c.RunCommand(fmt.Sprintf("mv -fv %s %s", remoteAttemptedStepsHashPath, remoteStepsHashPath))

	return nil
}
