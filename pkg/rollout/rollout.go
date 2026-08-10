package rollout

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/deploy"
	"github.com/initialed85/deployed/pkg/helpers/env"
	"github.com/initialed85/deployed/pkg/helpers/pointers"
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

	err = workspace.Validate(workspacePath)
	if err != nil {
		return err
	}

	if env.IsDebug {
		log.Printf("%s: ...\n\n%s", workspaceYAMLPath, workspace.PrettyFormat())
	}

	for _, mapping := range workspace.Mappings {
		log.Printf("deploying spec %#+v to %d pools", mapping.Spec.Name, len(mapping.Pools))

		for _, pool := range mapping.Pools {
			log.Printf("deploying spec %#+v to pool %#+v", mapping.Spec.Name, pool.Name)

			for _, target := range pool.Targets {
				log.Printf("deploying spec %#+v to target %#+v", mapping.Spec.Name, target)

				deployment := types.Deployment{
					ID:            uuid.Must(uuid.NewRandom()),
					ForceWithSudo: pointers.Ptr(true),
					Spec:          mapping.Spec,
					Target:        target,
				}

				tookAction, err := deploy.Deploy(deployment)
				if err != nil {
					return fmt.Errorf("failed to deploy spec %#+v to target %#+v", mapping.Spec.Name, target)
				}

				if tookAction {
					log.Printf("deployment of spec %#+v to target %#+v succeeded", mapping.Spec.Name, target)
				} else {
					log.Printf("deployment of spec %#+v to target %#+v was a no-op", mapping.Spec.Name, target)
				}
			}
		}
	}

	return nil
}
