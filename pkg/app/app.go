package app

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/initialed85/deployed/pkg/connection"
	"github.com/initialed85/deployed/pkg/rollout"
	"github.com/initialed85/deployed/pkg/types"
)

func App(versions ...string) error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: [ssh | ssh-with-sudo | deploy | rollout | version]")
	}

	switch os.Args[1] {

	case "ssh", "ssh-with-sudo":
		if len(os.Args) < 4 {
			return fmt.Errorf("usage: %s %s [target] [command ...]", os.Args[0], os.Args[1])
		}

		username, password, host, port, err := types.ParseTarget(os.Args[2])
		if err != nil {
			return err
		}

		c, err := connection.OpenSSH(host, port, username, password)
		if err != nil {
			return err
		}

		var out string

		switch os.Args[1] {

		case "ssh":
			out, _, err = c.RunCommand(strings.Join(os.Args[3:], " "))

		case "ssh-with-sudo":
			out, _, err = c.RunCommandWithSudo(strings.Join(os.Args[3:], " "))

		default:
			err = fmt.Errorf("assertion failed: this should not be possible")
		}

		if err != nil {
			return err
		}

		fmt.Print(out)

	case "deploy":
		return fmt.Errorf("TODO")

	case "rollout":
		if len(os.Args) < 3 {
			return fmt.Errorf("for %s %s, second argument must be path to workspace YAML", os.Args[0], os.Args[1])
		}

		err := rollout.Rollout(os.Args[2])
		if err != nil {
			return err
		}

	case "version":
		if len(versions) == 0 {
			return fmt.Errorf("app.App called without versions variadic")
		}

		log.Printf("%s: %s", os.Args[0], versions[0])

	default:
		return fmt.Errorf("unrecognized usage %s %s", os.Args[0], os.Args[1])
	}

	return nil
}
