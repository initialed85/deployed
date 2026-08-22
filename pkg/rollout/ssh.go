package rollout

import (
	"fmt"
	"strings"
	"sync"

	"github.com/initialed85/deployed/pkg/connection"
	"github.com/initialed85/deployed/pkg/helpers/env"
	"github.com/initialed85/deployed/pkg/types"
)

func SSH(workspaceYAMLPath string, poolName string, command string, withSudo bool) (string, string, error) {
	workspace, err := types.LoadWorkspace(workspaceYAMLPath, true, true, true)
	if err != nil {
		return "", "", err
	}

	if env.ForceWithSudo != nil {
		withSudo = *env.ForceWithSudo
	}

	wg := new(sync.WaitGroup)
	mu := new(sync.Mutex)
	var allStdout strings.Builder
	var allStderr strings.Builder
	errs := make([]error, 0)

	for _, pool := range workspace.Pools {
		if pool.GetName() != poolName {
			continue
		}

		for _, target := range pool.Targets {
			wg.Go(func() {
				stdout, stderr, err := func() (string, string, error) {
					username, password, host, port, err := types.ParseTarget(target)
					if err != nil {
						return "", "", err
					}

					c, err := connection.OpenSSH(host, port, username, password)
					if err != nil {
						return "", "", err
					}

					var stdout string
					var stderr string

					if !withSudo {
						stdout, stderr, err = c.RunCommand(command)
					} else {
						stdout, stderr, err = c.RunCommandWithSudo(command)
					}

					if err != nil {
						return "", "", err
					}

					return stdout, stderr, nil
				}()
				if err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					return
				}

				mu.Lock()
				if strings.TrimSpace(stdout) != "" {
					fmt.Fprintf(&allStdout, "---- %s ----\n\n", target)
					allStdout.WriteString(strings.TrimRight(stdout, "\n"))
					allStdout.WriteString("\n\n")
				}

				if strings.TrimSpace(stderr) != "" {
					fmt.Fprintf(&allStderr, "---- %s ----\n\n", target)
					allStderr.WriteString(strings.TrimRight(stderr, "\n"))
					allStderr.WriteString("\n\n")
				}
				mu.Unlock()
			})
		}
	}

	wg.Wait()

	if strings.TrimSpace(allStdout.String()) == "" {
		allStdout.Reset()
	}

	if strings.TrimSpace(allStderr.String()) == "" {
		allStderr.Reset()
	}

	if len(errs) > 0 {
		errMsgs := make([]string, 0)

		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}

		return allStdout.String(), allStderr.String(), fmt.Errorf("had %d errors while deploying...\n\n%s", len(errs), strings.Join(errMsgs, "\n"))
	}

	return allStdout.String(), allStderr.String(), nil
}
