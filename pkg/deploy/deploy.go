package deploy

import (
	"fmt"
	_log "log"
	"os"
	"strings"

	"github.com/initialed85/deployed/pkg/ssh"
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

	//
	// TODO(initialed85): extract Target + related handling into a struct
	//

	username, password, host, port, err := types.ParseTarget(deployment.Target)
	if err != nil {
		return false, err
	}

	c, err := ssh.Connect(host, int(port), username, password)
	if err != nil {
		return false, err
	}

	remoteHashPath := deployment.HashFile()

	out, _, _ := c.RunCommand(fmt.Sprintf("test -f '%s' && cat '%s'", remoteHashPath, remoteHashPath))

	remoteHash := strings.TrimSpace(out)

	if remoteHash == localHash {
		log.Printf("deploy no-op (local hash and remote hash match)")
		return false, nil
	}

	remoteAttemptedHashPath := fmt.Sprintf("%s.attempted-%s", deployment.HashFile(), deployment.ID)

	err = c.UploadFile(localHashPath, remoteAttemptedHashPath)
	if err != nil {
		return false, err
	}

	tookAction := false

	for i, step := range deployment.Spec.Steps {
		//
		// upload files / folders
		//

		err := func() error {
			for _, uploads := range step.Uploads {
				withSudo := false

				if deployment.WithSudo == nil {
					if uploads.WithSudo != nil {
						withSudo = *uploads.WithSudo
					} else if step.WithSudo != nil {
						withSudo = *step.WithSudo
					} else if deployment.Spec.WithSudo != nil {
						withSudo = *deployment.Spec.WithSudo
					}
				} else {
					withSudo = *deployment.WithSudo
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

			if deployment.WithSudo == nil {
				if step.WithSudo != nil {
					withSudo = *step.WithSudo
				} else if deployment.Spec.WithSudo != nil {
					withSudo = *deployment.Spec.WithSudo
				}
			} else {
				withSudo = *deployment.WithSudo
			}

			var runCommand func(string) (string, string, error)

			if withSudo {
				runCommand = c.RunCommandWithSudo
			} else {
				runCommand = c.RunCommand
			}

			localAndRemotePath := fmt.Sprintf("/tmp/deployed-%s-step-%d-script-%s.sh", deployment.Spec.Name, i, deployment.ID)

			err = os.WriteFile(localAndRemotePath, []byte(step.Script), 0o777)
			if err != nil {
				return err
			}

			defer func() {
				_ = os.Remove(localAndRemotePath)
			}()

			err = c.UploadFile(localAndRemotePath, localAndRemotePath)
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

		//
		// download files / folders
		//

		err = func() error {
			for _, downloads := range step.Downloads {
				withSudo := false

				if deployment.WithSudo == nil {
					if downloads.WithSudo != nil {
						withSudo = *downloads.WithSudo
					} else if step.WithSudo != nil {
						withSudo = *step.WithSudo
					} else if deployment.Spec.WithSudo != nil {
						withSudo = *deployment.Spec.WithSudo
					}
				} else {
					withSudo = *deployment.WithSudo
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

	_, _, err = c.RunCommand(fmt.Sprintf("mv -fv %s %s", remoteAttemptedHashPath, remoteHashPath))
	if err != nil {
		return tookAction, fmt.Errorf("failed to mark attempted steps as confirmed because %s", err)
	}

	log.Printf("deploy complete (took action)")

	return tookAction, nil
}
