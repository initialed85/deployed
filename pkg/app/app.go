package app

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/initialed85/deployed/pkg/connection"
	"github.com/initialed85/deployed/pkg/rollout"
	"github.com/initialed85/deployed/pkg/types"
)

func App(versions ...string) error {
	args := slices.Clone(os.Args)

	if len(args) > 1 && (args[1] == "-" || args[1] == "--") {
		args = args[1:]
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: [ssh | ssh-with-sudo | ssh-to-pool | ssh-with-sudo-to-pool | deploy | rollout | version]")
	}

	switch args[1] {

	case "ssh", "ssh-with-sudo":
		if len(args) < 4 {
			return fmt.Errorf("usage: %s %s [target] [command ...]", args[0], args[1])
		}

		username, password, host, port, err := types.ParseTarget(args[2])
		if err != nil {
			return err
		}

		c, err := connection.OpenSSH(host, port, username, password)
		if err != nil {
			return err
		}

		var out string

		switch args[1] {

		case "ssh":
			out, _, err = c.RunCommand(strings.Join(args[3:], " "))

		case "ssh-with-sudo":
			out, _, err = c.RunCommandWithSudo(strings.Join(args[3:], " "))

		default:
			err = fmt.Errorf("assertion failed: unhandled command %#+v; this should not be possible", args[1])
		}

		if err != nil {
			return err
		}

		fmt.Print(out)

	case "ssh-to-pool", "ssh-with-sudo-to-pool":
		if len(args) < 5 {
			return fmt.Errorf("usage: %s %s [path to workspace YAML] [pool name] [command ...]", args[0], args[1])
		}

		var stdout string
		var stderr string
		var err error

		switch args[1] {

		case "ssh-to-pool":
			stdout, stderr, err = rollout.SSH(args[2], args[3], strings.Join(args[4:], " "), false)

		case "ssh-with-sudo-to-pool":
			stdout, stderr, err = rollout.SSH(args[2], args[3], strings.Join(args[4:], " "), true)

		default:
			err = fmt.Errorf("assertion failed: unhandled command %#+v; this should not be possible", args[1])
		}

		if err != nil {
			return err
		}

		if strings.TrimSpace(stdout) != "" {
			fmt.Print("#\n")
			fmt.Print("# stdout\n")
			fmt.Print("#\n\n")
			fmt.Print(stdout)
		}

		if strings.TrimSpace(stderr) != "" {
			fmt.Print("#\n")
			fmt.Print("# stderr\n")
			fmt.Print("#\n\n")
			fmt.Print(stderr)
		}

	case "deploy":
		return fmt.Errorf("TODO")

	case "rollout":
		if len(args) < 3 {
			return fmt.Errorf("usage: %s %s [path to workspace YAML]", args[0], args[1])
		}

		err := rollout.Rollout(args[2])
		if err != nil {
			return err
		}

	case "version":
		if len(versions) == 0 {
			return fmt.Errorf("app.App called without versions variadic")
		}

		log.Printf("%s: %s", args[0], versions[0])

	default:
		return fmt.Errorf("unrecognized usage %s %s", args[0], args[1])
	}

	return nil
}
