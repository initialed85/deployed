package types

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStepValidateResolvesRelativeUploadPath(t *testing.T) {
	root := t.TempDir()
	localPath := filepath.Join(root, "payload.txt")
	require.NoError(t, os.WriteFile(localPath, []byte("payload\n"), 0o644))

	step := Step{
		Uploads: []Upload{{Local: "payload.txt"}},
	}

	require.NoError(t, step.Validate(root))
	require.Equal(t, localPath, step.Uploads[0].Local)
}

func TestStepValidateResolvesRelativeDownloadPath(t *testing.T) {
	root := t.TempDir()
	localPath := filepath.Join(root, "download.txt")

	step := Step{
		Downloads: []Download{{Local: "download.txt"}},
	}

	require.NoError(t, step.Validate(root))
	require.Equal(t, localPath, step.Downloads[0].Local)
}

func TestHashesChangeWhenUploadedFileChanges(t *testing.T) {
	localPath := filepath.Join(t.TempDir(), "payload.txt")
	require.NoError(t, os.WriteFile(localPath, []byte("old content\n"), 0o644))

	deployment := Deployment{
		Spec: &Spec{
			Name: "spec",
			Steps: []Step{{
				Uploads: []Upload{{
					Local:  localPath,
					Remote: "/etc/haproxy.cfg",
				}},
			}},
		},
		Target: "user:pass@example.com:22",
	}

	stepHashBefore, err := deployment.Spec.Steps[0].Hash()
	require.NoError(t, err)
	deploymentHashBefore, err := deployment.Hash()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(localPath, []byte("new content\n"), 0o644))

	stepHashAfter, err := deployment.Spec.Steps[0].Hash()
	require.NoError(t, err)
	deploymentHashAfter, err := deployment.Hash()
	require.NoError(t, err)

	require.NotEqual(t, stepHashBefore, stepHashAfter)
	require.NotEqual(t, deploymentHashBefore, deploymentHashAfter)
}

func TestStepHashChangesWhenDefinitionChanges(t *testing.T) {
	localPath := filepath.Join(t.TempDir(), "payload.txt")
	require.NoError(t, os.WriteFile(localPath, []byte("content\n"), 0o644))

	tests := []struct {
		name   string
		mutate func(*Step)
	}{
		{
			name: "script",
			mutate: func(step *Step) {
				step.Script = "printf 'changed\\n'"
			},
		},
		{
			name: "upload remote path",
			mutate: func(step *Step) {
				step.Uploads[0].Remote = "/etc/other.conf"
			},
		},
		{
			name: "step sudo",
			mutate: func(step *Step) {
				withSudo := true
				step.WithSudo = &withSudo
			},
		},
		{
			name: "download remote path",
			mutate: func(step *Step) {
				step.Downloads[0].Remote = "/var/lib/other.conf"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := Step{
				Uploads: []Upload{{
					Local:  localPath,
					Remote: "/etc/example.conf",
				}},
				Downloads: []Download{{
					Local:  filepath.Join(t.TempDir(), "download.txt"),
					Remote: "/var/lib/example.conf",
				}},
			}

			hashBefore, err := step.Hash()
			require.NoError(t, err)

			tt.mutate(&step)

			hashAfter, err := step.Hash()
			require.NoError(t, err)
			require.NotEqual(t, hashBefore, hashAfter, fmt.Sprintf("test case: %s", tt.name))
		})
	}
}
