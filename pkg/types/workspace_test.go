package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceValidateResolvesRelativeTransferPaths(t *testing.T) {
	root := t.TempDir()
	localPath := filepath.Join(root, "payload.txt")
	require.NoError(t, os.WriteFile(localPath, []byte("payload\n"), 0o644))

	workspace := &Workspace{
		Specs: []*Spec{{
			Name: "spec",
			Steps: []Step{{
				Uploads:   []Upload{{Local: "payload.txt"}},
				Downloads: []Download{{Local: "download.txt"}},
			}},
		}},
		Pools: []*Pool{{
			Name:    "pool",
			Targets: []string{"user:pass@example.com:22"},
		}},
	}

	require.NoError(t, workspace.Validate(root))
	require.Equal(t, localPath, workspace.Specs[0].Steps[0].Uploads[0].Local)
	require.Equal(t, filepath.Join(root, "download.txt"), workspace.Specs[0].Steps[0].Downloads[0].Local)
}

func TestWorkspaceValidateDoesNotMutatePoolTargetsAcrossMappings(t *testing.T) {
	root := t.TempDir()
	target := "user:pass@example.com:22"

	workspace := &Workspace{
		Specs: []*Spec{{
			Name:  "spec",
			Steps: []Step{{}},
		}},
		Pools: []*Pool{
			{Name: "first", Targets: []string{target}},
			{Name: "second", Targets: []string{target}},
		},
		Mappings: []Mapping{
			{SpecNames: []string{"spec"}, PoolNames: []string{"first", "second"}},
			{SpecNames: []string{"spec"}, PoolNames: []string{"second"}},
		},
	}

	require.NoError(t, workspace.Validate(root))
	require.Equal(t, []string{target}, workspace.Mappings[0].Pools[0].Targets)
	require.Empty(t, workspace.Mappings[0].Pools[1].Targets)
	require.Equal(t, []string{target}, workspace.Mappings[1].Pools[0].Targets)
}

func TestLoadWorkspaceUsesWorkspaceDirectoryForRelativeScripts(t *testing.T) {
	root := t.TempDir()
	script := "#!/bin/sh\nprintf 'script content\\n'\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "script.sh"), []byte(script), 0o644))

	workspacePath := filepath.Join(root, "workspace.yaml")
	workspaceYAML := `specs:
  - name: spec
    steps:
      - script: "@script.sh"
pools:
  - name: pool
    targets:
      - user:pass@example.com:22
mappings:
  - specs: [spec]
    pools: [pool]
`
	require.NoError(t, os.WriteFile(workspacePath, []byte(workspaceYAML), 0o644))

	workingDirectory, err := os.Getwd()
	require.NoError(t, err)

	workspace, err := LoadWorkspace(workspacePath, true, true, true)
	require.NoError(t, err)
	require.Equal(t, script, workspace.Specs[0].Steps[0].Script)

	currentWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, workingDirectory, currentWorkingDirectory)
}
