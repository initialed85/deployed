package app

import (
	"fmt"
	"os"

	"github.com/initialed85/deployed/pkg/rollout"
)

func App() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: [ssh | deploy | rollout]")
	}

	switch os.Args[1] {

	case "ssh":
		if len(os.Args) != 5 {
			return fmt.Errorf("usage: %s [target] [run_command | run_command_with_sudo | upload | download] [command or path]", os.Args[1])
		}

		return fmt.Errorf("TODO")

	case "deploy":
		if len(os.Args) != 5 {
			return fmt.Errorf("usage: %s [path to workspace YAML] [name of spec] [name of target]", os.Args[1])
		}

		return fmt.Errorf("TODO")

	case "rollout":
		if len(os.Args) < 3 {
			return fmt.Errorf("for %#+v, second argument must be path to workspace YAML", os.Args[1])
		}

		err := rollout.Rollout(os.Args[2])
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unrecognized command %#+v", os.Args[1])
	}

	return nil
}
