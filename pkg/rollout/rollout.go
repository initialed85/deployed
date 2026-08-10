package rollout

import (
	"os"
	"path/filepath"

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

	return nil
}
