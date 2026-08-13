package deploy

import (
	"fmt"
	_log "log"
	"os"
	"strings"

	"github.com/initialed85/deployed/pkg/connection"
	"github.com/initialed85/deployed/pkg/connection/connection_types"
	"github.com/initialed85/deployed/pkg/types"
)

func Deploy(deployment types.Deployment) (bool, error) {
	deployment.Validate() // sets deployment ID etc

	log := _log.New(
		os.Stdout,
		fmt.Sprintf("Deploy{%s, %s} ", deployment.Target, deployment.ID),
		_log.Ldate|_log.Ltime|_log.Lmicroseconds|_log.LUTC|_log.Lmsgprefix,
	)

	log.Printf("starting deploy")

	localHash, err := deployment.Hash()
	if err != nil {
		return false, err
	}

	log.Printf("local hash: %s", localHash)

	localHashPath := fmt.Sprintf("/tmp/%s.local-%s", deployment.HashFile(), deployment.ID)

	err = os.WriteFile(localHashPath, []byte(localHash), 0o777)
	if err != nil {
		return false, err
	}

	defer func() {
		_ = os.Remove(localHashPath)
	}()

	username, password, host, port, err := types.ParseTarget(deployment.Target)
	if err != nil {
		return false, err
	}

	open := func() connection_types.OpenDeployableFn {
		if port <= 0 {
			return connection.OpenLocal
		} else {
			return connection.OpenSSH
		}
	}()

	c, err := open(host, port, username, password)
	if err != nil {
		return false, err
	}

	defer c.Close()

	remoteHashPath := deployment.HashFile()

	out, _, _ := c.RunCommand(fmt.Sprintf("test -f '%s' && cat '%s'", remoteHashPath, remoteHashPath))

	remoteHash := strings.TrimSpace(out)

	remoteAttemptedHashPath := fmt.Sprintf("%s.attempted-%s", deployment.HashFile(), deployment.ID)

	if remoteHash != localHash {
		err = c.Upload(localHashPath, remoteAttemptedHashPath)
		if err != nil {
			return false, err
		}
	}

	tookAction := false

	for i, step := range deployment.Spec.Steps {
		if remoteHash != localHash {
			//
			// upload files / folders
			//

			err := func() error {

				for _, uploads := range step.Uploads {
					withSudo := false

					if deployment.ForceWithSudo == nil {
						if uploads.WithSudo != nil {
							withSudo = *uploads.WithSudo
						} else if step.WithSudo != nil {
							withSudo = *step.WithSudo
						} else if deployment.Spec.WithSudo != nil {
							withSudo = *deployment.Spec.WithSudo
						}
					} else {
						withSudo = *deployment.ForceWithSudo
					}

					var upload func(string, string) error

					if withSudo {
						upload = c.UploadWithSudo
					} else {
						upload = c.Upload
					}

					err = upload(uploads.Local, uploads.Remote)
					if err != nil {
						return err
					}

					tookAction = true
				}

				return nil
			}()
			if err != nil {
				return tookAction, fmt.Errorf("failed to execute step %d uploads because %s", i, err)
			}

			//
			// run scripts
			//

			err = func() error {
				if step.Script == "" {
					return nil
				}

				withSudo := false

				if deployment.ForceWithSudo == nil {
					if step.WithSudo != nil {
						withSudo = *step.WithSudo
					} else if deployment.Spec.WithSudo != nil {
						withSudo = *deployment.Spec.WithSudo
					}
				} else {
					withSudo = *deployment.ForceWithSudo
				}

				var runCommand func(string) (string, string, error)

				if withSudo {
					runCommand = c.RunCommandWithSudo
				} else {
					runCommand = c.RunCommand
				}

				localAndRemotePath := fmt.Sprintf("/tmp/deployed-%s-step-%d-script-%s.sh", deployment.Spec.GetName(), i, deployment.ID)

				err = os.WriteFile(localAndRemotePath, []byte(step.Script), 0o777)
				if err != nil {
					return err
				}

				defer func() {
					_ = os.Remove(localAndRemotePath)
				}()

				err = c.Upload(localAndRemotePath, localAndRemotePath)
				if err != nil {
					return err
				}

				defer func() {
					_, _, _ = c.RunCommand(fmt.Sprintf("rm -f '%s' || true", localAndRemotePath))
				}()

				tookAction = true

				_, _, err := runCommand(localAndRemotePath)
				if err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				return tookAction, fmt.Errorf("failed to execute step %d script because %s", i, err)
			}
		}

		//
		// download files / folders
		//

		err = func() error {
			for _, downloads := range step.Downloads {
				withSudo := false

				if deployment.ForceWithSudo == nil {
					if downloads.WithSudo != nil {
						withSudo = *downloads.WithSudo
					} else if step.WithSudo != nil {
						withSudo = *step.WithSudo
					} else if deployment.Spec.WithSudo != nil {
						withSudo = *deployment.Spec.WithSudo
					}
				} else {
					withSudo = *deployment.ForceWithSudo
				}

				var download func(string, string) error

				if withSudo {
					download = c.DownloadWithSudo
				} else {
					download = c.Download
				}

				err = download(downloads.Remote, downloads.Local)
				if err != nil {
					return err
				}
			}

			return nil
		}()
		if err != nil {
			return tookAction, fmt.Errorf("failed to execute step %d downloads because %s", i, err)
		}
	}

	if remoteHash == localHash {
		log.Printf("deploy no-op (local hash and remote hash match)")
		return false, nil
	}

	_, _, err = c.RunCommand(fmt.Sprintf("mv -fv %s %s", remoteAttemptedHashPath, remoteHashPath))
	if err != nil {
		return tookAction, fmt.Errorf("failed to mark attempted steps as confirmed because %s", err)
	}

	log.Printf("deploy complete (took action)")

	return tookAction, nil
}
