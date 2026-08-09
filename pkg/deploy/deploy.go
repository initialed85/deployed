package deploy

import (
	"fmt"
	_log "log"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/ssh"
	"github.com/initialed85/deployed/pkg/types"
)

func Deploy(target string, steps []types.Step) (bool, error) {
	deployID := uuid.Must(uuid.NewRandom())

	log := _log.New(
		os.Stdout,
		fmt.Sprintf("Deploy{%s, %s} ", target, deployID),
		_log.Ldate|_log.Ltime|_log.Lmicroseconds|_log.LUTC|_log.Lmsgprefix,
	)

	log.Printf("starting deploy")

	localStepsHash, err := types.HashSteps(steps)
	if err != nil {
		return false, err
	}

	log.Printf("local steps hash: %s", localStepsHash)

	localStepsHashPath := fmt.Sprintf("/tmp/deployed-%s-steps-hash.txt", deployID)

	err = os.WriteFile(localStepsHashPath, []byte(localStepsHash), 0o777)
	if err != nil {
		return false, err
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
		return false, fmt.Errorf("failed to split %#+v into (username:password)@(host:port) because %s", target, err)
	}

	c, err := ssh.Connect(host, int(port), username, password)
	if err != nil {
		return false, err
	}

	remoteStepsHashPath := "deployed-steps-hash.txt"

	out, _, _ := c.RunCommand(fmt.Sprintf("test -f '%s' && cat '%s'", remoteStepsHashPath, remoteStepsHashPath))

	remoteStepsHash := strings.TrimSpace(out)

	if remoteStepsHash == localStepsHash {
		log.Printf("local steps hash and remote steps hash match")
		log.Printf("deploy complete (no action required)")
		return false, nil
	}

	remoteAttemptedStepsHashPath := fmt.Sprintf("deployed-steps-hash.txt.attempted-%s", deployID)

	err = c.SendFile(localStepsHashPath, remoteAttemptedStepsHashPath)
	if err != nil {
		return false, err
	}

	pathsToCleanUp := make([]string, 0)

	tookAction := false

	for i, step := range steps {
		err := func() error {
			if step.Script != "" {
				path := fmt.Sprintf("/tmp/deployed-%s-step-%d-script.sh", deployID, i)

				err = os.WriteFile(path, []byte(step.Script), 0o777)
				if err != nil {
					return err
				}

				err = c.SendFile(path, path)
				if err != nil {
					return err
				}

				tookAction = true

				pathsToCleanUp = append(pathsToCleanUp, path)

				_, _, err := c.RunCommand(path)
				if err != nil {
					return err
				}
			}

			return nil
		}()
		if err != nil {
			return tookAction, fmt.Errorf("failed to execute step %d script because %s", i, err)
		}

		err = func() error {
			if step.ScriptWithSudo != "" {
				path := fmt.Sprintf("/tmp/deployed-%s-step-%d-script.sh", deployID, i)

				err = os.WriteFile(path, []byte(step.ScriptWithSudo), 0o777)
				if err != nil {
					return err
				}

				err = c.SendFile(path, path)
				if err != nil {
					return err
				}

				tookAction = true

				pathsToCleanUp = append(pathsToCleanUp, path)

				_, _, err := c.RunCommandWithSudo(path)
				if err != nil {
					return err
				}
			}

			return nil
		}()
		if err != nil {
			return tookAction, fmt.Errorf("failed to execute step %d script_with_sudo because %s", i, err)
		}
	}

	for _, path := range pathsToCleanUp {
		_, _, _ = c.RunCommand(fmt.Sprintf("rm -f %s || true", path))
	}

	_, _, err = c.RunCommand(fmt.Sprintf("mv -fv %s %s", remoteAttemptedStepsHashPath, remoteStepsHashPath))
	if err != nil {
		return tookAction, fmt.Errorf("failed to mark attempted steps as confirmed because %s", err)
	}

	log.Printf("deploy complete (took action)")

	return tookAction, nil
}
