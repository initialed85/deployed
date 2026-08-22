package rollout

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/deploy"
	"github.com/initialed85/deployed/pkg/helpers/env"
	"github.com/initialed85/deployed/pkg/types"
)

func Rollout(workspaceYAMLPath string) error {
	workspace, err := types.LoadWorkspace(workspaceYAMLPath, true, true, true)
	if err != nil {
		return err
	}

	wg := new(sync.WaitGroup)
	mu := new(sync.Mutex)
	errs := make([]error, 0)

	log.Printf("deploying to %d mappings", len(workspace.Mappings))

	for i, mapping := range workspace.Mappings {
		log.Printf("deploying mapping %d", i)

		for _, spec := range mapping.Specs {
			log.Printf("deploying spec %#+v to %d pools", spec.GetName(), len(mapping.Pools))

			for _, pool := range mapping.Pools {
				log.Printf("deploying spec %#+v to pool %#+v", spec.GetName(), pool.GetName())

				for _, target := range pool.Targets {
					log.Printf("deploying spec %#+v to target %#+v", spec.GetName(), target)

					deployment := types.Deployment{
						ID:            uuid.Must(uuid.NewRandom()),
						ForceWithSudo: env.ForceWithSudo,
						Spec:          spec,
						Target:        target,
					}

					wg.Go(func() {
						err := func() error {
							tookAction, err := deploy.Deploy(deployment)
							if err != nil {
								return fmt.Errorf("failed to deploy spec %#+v to target %#+v because %s", spec.GetName(), target, err)
							}

							if tookAction {
								log.Printf("deployment of spec %#+v to target %#+v succeeded", spec.GetName(), target)
							} else {
								log.Printf("deployment of spec %#+v to target %#+v was a no-op", spec.GetName(), target)
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
