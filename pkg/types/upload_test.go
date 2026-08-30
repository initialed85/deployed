package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUploadHashChangesWhenNestedFileChanges(t *testing.T) {
	root := t.TempDir()
	localPath := filepath.Join(root, "nested", "deeper", "payload.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(localPath), 0o755))
	require.NoError(t, os.WriteFile(localPath, []byte("old content\n"), 0o644))

	upload := Upload{Local: root}
	hashBefore, err := upload.Hash()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(localPath, []byte("new content\n"), 0o644))

	hashAfter, err := upload.Hash()
	require.NoError(t, err)
	require.NotEqual(t, hashBefore, hashAfter)
}

func TestUploadHashChangesWhenNestedFileIsRenamed(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "nested", "old-name.txt")
	newPath := filepath.Join(root, "nested", "new-name.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(oldPath), 0o755))
	require.NoError(t, os.WriteFile(oldPath, []byte("same content\n"), 0o644))

	upload := Upload{Local: root}
	hashBefore, err := upload.Hash()
	require.NoError(t, err)

	require.NoError(t, os.Rename(oldPath, newPath))

	hashAfter, err := upload.Hash()
	require.NoError(t, err)
	require.NotEqual(t, hashBefore, hashAfter)
}
