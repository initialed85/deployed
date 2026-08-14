package rollout

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/deploy"
	"github.com/initialed85/deployed/pkg/helpers/env"
	"github.com/initialed85/deployed/pkg/types"
	yaml "go.yaml.in/yaml/v4"
)

func Rollout(workspaceYAMLPath string) error {
	workspaceYAMLPath, err := filepath.Abs(workspaceYAMLPath)
	if err != nil {
		return err
	}

	rawWorkspace, err := os.ReadFile(workspaceYAMLPath)
	if err != nil {
		return err
	}

	var workspace types.Workspace
	err = yaml.Unmarshal(rawWorkspace, &workspace)
	if err != nil {
		return err
	}

	workspacePath, _ := filepath.Split(workspaceYAMLPath)

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	err = os.Chdir(workspacePath)
	if err != nil {
		return err
	}

	defer func() {
		_ = os.Chdir(wd)
	}()

	err = workspace.Validate(workspacePath)
	if err != nil {
		return err
	}

	if env.IsDebug {
		log.Printf("%s: ...\n\n%s", workspaceYAMLPath, workspace.PrettyFormat())
	}

	wg := new(sync.WaitGroup)
	mu := new(sync.Mutex)
	errs := make([]error, 0)

	for _, mapping := range workspace.Mappings {
		log.Printf("deploying spec %#+v to %d pools", mapping.Spec.GetName(), len(mapping.Pools))

		for _, pool := range mapping.Pools {
			log.Printf("deploying spec %#+v to pool %#+v", mapping.Spec.GetName(), pool.GetName())

			for _, target := range pool.Targets {
				log.Printf("deploying spec %#+v to target %#+v", mapping.Spec.GetName(), target)

				deployment := types.Deployment{
					ID:            uuid.Must(uuid.NewRandom()),
					ForceWithSudo: env.ForceWithSudo,
					Spec:          mapping.Spec,
					Target:        target,
				}

				wg.Go(func() {
					err := func() error {
						tookAction, err := deploy.Deploy(deployment)
						if err != nil {
							return fmt.Errorf("failed to deploy spec %#+v to target %#+v because %s", mapping.Spec.GetName(), target, err)
						}

						if tookAction {
							log.Printf("deployment of spec %#+v to target %#+v succeeded", mapping.Spec.GetName(), target)
						} else {
							log.Printf("deployment of spec %#+v to target %#+v was a no-op", mapping.Spec.GetName(), target)
						}

						return nil
					}()
					if err != nil {
						mu.Lock()
						errs = append(errs, err)
						mu.Unlock()
					}
				})
			}
		}

		wg.Wait()
	}

	wg.Wait()

	if len(errs) > 0 {
		errMsgs := make([]string, 0)

		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}

		return fmt.Errorf("had %d errors while deploying...\n\n%s", len(errs), strings.Join(errMsgs, "\n"))
	}

	return nil
}
