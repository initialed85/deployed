package types

import (
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
