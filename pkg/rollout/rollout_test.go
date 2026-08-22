package rollout

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/initialed85/deployed/pkg/connection"
	"github.com/stretchr/testify/require"
)

func TestRollout(t *testing.T) {
	err := Rollout("../../test/k3s.yaml")
	require.NoError(t, err)
}

func TestRolloutBrokenYAML(t *testing.T) {
	fixtures, err := filepath.Glob("../../test/rollout-modes/*-broken.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, fixtures)

	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			err := Rollout(fixture)
			require.Error(t, err)
		})
	}
}

func TestRolloutFolderMerge(t *testing.T) {
	const remoteDir = "/home/user2/deployed-rollout-modes/folder-merge"

	c, err := connection.OpenSSH("localhost", 2221, "user2", "Password2")
	require.NoError(t, err)
	defer c.Close()

	cleanup := fmt.Sprintf("rm -rf '%s' deployed-deployment-rollout-modes-folder-merge* deployed-step-rollout-modes-folder-merge*", remoteDir)
	_, _, err = c.RunCommand(cleanup)
	require.NoError(t, err)
	defer func() { _, _, _ = c.RunCommand(cleanup) }()

	_, _, err = c.RunCommand(fmt.Sprintf("mkdir -p '%s/nested/deep' && printf 'old root content\\n' > '%s/root-overwrite.txt' && printf 'keep root content\\n' > '%s/root-existing.txt' && printf 'old nested content\\n' > '%s/nested/overwrite.txt' && printf 'keep nested content\\n' > '%s/nested/existing.txt' && printf 'keep deep content\\n' > '%s/nested/deep/existing.txt'", remoteDir, remoteDir, remoteDir, remoteDir, remoteDir, remoteDir))
	require.NoError(t, err)

	fixturePath, err := filepath.Abs("../../test/rollout-modes/folder-merge.yaml")
	require.NoError(t, err)
	fixture, err := os.ReadFile(fixturePath)
	require.NoError(t, err)
	workspacePath := filepath.Join(t.TempDir(), "workspace.yaml")
	localDownloadFolder := filepath.Join(filepath.Dir(workspacePath), "folder-merge-download")
	fixture = []byte(strings.NewReplacer(
		"local: folder-merge-payload", fmt.Sprintf("local: %q", filepath.Join(filepath.Dir(fixturePath), "folder-merge-payload")),
		"local: folder-merge-download", fmt.Sprintf("local: %q", localDownloadFolder),
	).Replace(string(fixture)))
	err = os.WriteFile(workspacePath, fixture, 0o644)
	require.NoError(t, err)

	err = Rollout(workspacePath)
	require.NoError(t, err)

	files := map[string]string{
		"root-overwrite.txt":       "new root content\n",
		"root-new.txt":             "new root file\n",
		"root-existing.txt":        "keep root content\n",
		"nested/overwrite.txt":     "new nested content\n",
		"nested/new.txt":           "new nested file\n",
		"nested/existing.txt":      "keep nested content\n",
		"nested/deep/new.txt":      "new deep file\n",
		"nested/deep/existing.txt": "keep deep content\n",
	}

	for path, expected := range files {
		stdout, _, err := c.RunCommand(fmt.Sprintf("cat '%s/%s'", remoteDir, path))
		require.NoError(t, err, path)
		require.Equal(t, expected, stdout, path)

		contents, err := os.ReadFile(filepath.Join(localDownloadFolder, path))
		require.NoError(t, err, path)
		require.Equal(t, expected, string(contents), path)
	}
}

func TestRolloutLocalDownloads(t *testing.T) {
	fixturePath, err := filepath.Abs("../../test/rollout-modes/local-downloads.yaml")
	require.NoError(t, err)

	workspaceDir := t.TempDir()
	workspacePath := filepath.Join(workspaceDir, "workspace.yaml")
	remoteFile := filepath.Join(workspaceDir, "remote-file.txt")
	localFile := filepath.Join(workspaceDir, "local-file.txt")
	remoteFolder := filepath.Join(workspaceDir, "remote-folder")
	localFolder := filepath.Join(workspaceDir, "local-folder")

	writeFile := func(path, contents string) {
		err := os.MkdirAll(filepath.Dir(path), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(contents), 0o644)
		require.NoError(t, err)
	}

	writeFile(remoteFile, "downloaded file content\n")
	writeFile(filepath.Join(remoteFolder, "root-overwrite.txt"), "new root content\n")
	writeFile(filepath.Join(remoteFolder, "root-new.txt"), "new root file\n")
	writeFile(filepath.Join(remoteFolder, "nested", "overwrite.txt"), "new nested content\n")
	writeFile(filepath.Join(remoteFolder, "nested", "new.txt"), "new nested file\n")
	writeFile(filepath.Join(remoteFolder, "nested", "deep", "new.txt"), "new deep file\n")
	writeFile(filepath.Join(localFolder, "root-overwrite.txt"), "old root content\n")
	writeFile(filepath.Join(localFolder, "root-existing.txt"), "keep root content\n")
	writeFile(filepath.Join(localFolder, "nested", "overwrite.txt"), "old nested content\n")
	writeFile(filepath.Join(localFolder, "nested", "existing.txt"), "keep nested content\n")
	writeFile(filepath.Join(localFolder, "nested", "deep", "existing.txt"), "keep deep content\n")

	fixture, err := os.ReadFile(fixturePath)
	require.NoError(t, err)
	fixture = []byte(strings.NewReplacer(
		"remote: local-download-file-source", fmt.Sprintf("remote: %q", remoteFile),
		"local: local-download-file-destination", fmt.Sprintf("local: %q", localFile),
		"remote: local-download-folder-source", fmt.Sprintf("remote: %q", remoteFolder),
		"local: local-download-folder-destination", fmt.Sprintf("local: %q", localFolder),
	).Replace(string(fixture)))
	err = os.WriteFile(workspacePath, fixture, 0o644)
	require.NoError(t, err)

	c, err := connection.OpenLocal("__local__", 0, "${USER}", "${PASS}")
	require.NoError(t, err)
	defer c.Close()

	cleanup := "rm -f deployed-deployment-rollout-modes-local-downloads* deployed-step-rollout-modes-local-downloads*"
	_, _, err = c.RunCommand(cleanup)
	require.NoError(t, err)
	defer func() { _, _, _ = c.RunCommand(cleanup) }()

	err = Rollout(workspacePath)
	require.NoError(t, err)

	contents, err := os.ReadFile(localFile)
	require.NoError(t, err)
	require.Equal(t, "downloaded file content\n", string(contents))

	files := map[string]string{
		"root-overwrite.txt":       "new root content\n",
		"root-new.txt":             "new root file\n",
		"root-existing.txt":        "keep root content\n",
		"nested/overwrite.txt":     "new nested content\n",
		"nested/new.txt":           "new nested file\n",
		"nested/existing.txt":      "keep nested content\n",
		"nested/deep/new.txt":      "new deep file\n",
		"nested/deep/existing.txt": "keep deep content\n",
	}

	for path, expected := range files {
		contents, err := os.ReadFile(filepath.Join(localFolder, path))
		require.NoError(t, err, path)
		require.Equal(t, expected, string(contents), path)
	}
}

func TestRolloutModes(t *testing.T) {
	modes := []struct {
		name      string
		fixture   string
		specName  string
		remoteDir string
		upload    bool
		script    bool
		download  bool
	}{
		{name: "Full", fixture: "full.yaml", specName: "rollout-modes-full", remoteDir: "/home/user2/deployed-rollout-modes/full", upload: true, script: true, download: true},
		{name: "ScriptsOnly", fixture: "scripts-only.yaml", specName: "rollout-modes-scripts-only", remoteDir: "/home/user2/deployed-rollout-modes/scripts-only", script: true},
		{name: "UploadsOnly", fixture: "uploads-only.yaml", specName: "rollout-modes-uploads-only", remoteDir: "/home/user2/deployed-rollout-modes/uploads-only", upload: true},
		{name: "DownloadsOnly", fixture: "downloads-only.yaml", specName: "rollout-modes-downloads-only", remoteDir: "/home/user2/deployed-rollout-modes/downloads-only", download: true},
	}

	for _, mode := range modes {
		mode := mode
		t.Run(mode.name, func(t *testing.T) {
			workspaceDir := t.TempDir()
			fixtureDir := "../../test/rollout-modes"
			workspacePath := filepath.Join(workspaceDir, "workspace.yaml")
			localDownload := filepath.Join(workspaceDir, "download.txt")
			remoteUpload := fmt.Sprintf("%s/upload.txt", mode.remoteDir)
			remoteScript := fmt.Sprintf("%s/script.txt", mode.remoteDir)
			remoteDownload := fmt.Sprintf("%s/download.txt", mode.remoteDir)

			fixture, err := os.ReadFile(filepath.Join(fixtureDir, mode.fixture))
			require.NoError(t, err)
			fixture = []byte(strings.NewReplacer(
				"local: download.txt", fmt.Sprintf("local: %q", localDownload),
				"local: payload.txt", fmt.Sprintf("local: %q", filepath.Join(workspaceDir, "payload.txt")),
			).Replace(string(fixture)))
			err = os.WriteFile(workspacePath, fixture, 0o644)
			require.NoError(t, err)

			payload, err := os.ReadFile(filepath.Join(fixtureDir, "payload.txt"))
			require.NoError(t, err)
			err = os.WriteFile(filepath.Join(workspaceDir, "payload.txt"), payload, 0o644)
			require.NoError(t, err)

			c, err := connection.OpenSSH("localhost", 2221, "user2", "Password2")
			require.NoError(t, err)
			defer c.Close()

			cleanup := fmt.Sprintf("rm -rf '%s' deployed-deployment-%s* deployed-step-%s*", mode.remoteDir, mode.specName, mode.specName)
			_, _, err = c.RunCommand(cleanup)
			require.NoError(t, err)
			defer func() { _, _, _ = c.RunCommand(cleanup) }()

			_, _, err = c.RunCommand(fmt.Sprintf("mkdir -p '%s'", mode.remoteDir))
			require.NoError(t, err)

			if mode.download {
				_, _, err = c.RunCommand(fmt.Sprintf("printf 'download content\\n' > '%s'", remoteDownload))
				require.NoError(t, err)
			}

			err = Rollout(workspacePath)
			require.NoError(t, err)

			if mode.upload {
				stdout, _, err := c.RunCommand(fmt.Sprintf("cat '%s'", remoteUpload))
				require.NoError(t, err)
				require.Equal(t, "upload content\n", stdout)
			}

			if mode.script {
				stdout, _, err := c.RunCommand(fmt.Sprintf("cat '%s'", remoteScript))
				require.NoError(t, err)
				require.Equal(t, "script content\n", stdout)
			}

			if mode.download {
				contents, err := os.ReadFile(localDownload)
				require.NoError(t, err)
				require.Equal(t, "download content\n", string(contents))
			}

			if mode.upload {
				_, _, err = c.RunCommand(fmt.Sprintf("printf 'changed content\\n' > '%s'", remoteUpload))
				require.NoError(t, err)
			}

			if mode.download {
				err = os.Remove(localDownload)
				require.NoError(t, err)
			}

			err = Rollout(workspacePath)
			require.NoError(t, err)

			if mode.upload {
				stdout, _, err := c.RunCommand(fmt.Sprintf("cat '%s'", remoteUpload))
				require.NoError(t, err)
				require.Equal(t, "changed content\n", stdout)
			}

			if mode.download {
				contents, err := os.ReadFile(localDownload)
				require.NoError(t, err)
				require.Equal(t, "download content\n", string(contents))
			}
		})
	}
}
