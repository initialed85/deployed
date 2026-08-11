package types

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func ParseTarget(target string) (string, string, string, int64, error) {
	var username, password, host string
	var port int64

	err := func() error {
		parts := strings.Split(target, "@")
		if len(parts) < 2 {
			return fmt.Errorf("target doesn't have '@' symbol")
		}

		usernameAndPassword := parts[0]
		usernameAndPasswordParts := strings.Split(usernameAndPassword, ":")
		if len(usernameAndPasswordParts) < 2 {
			return fmt.Errorf("username-and-password (left) half doesn't have ':' symbol")
		}
		username = usernameAndPasswordParts[0]
		password = usernameAndPasswordParts[1]

		hostAndPort := parts[1]
		hostAndPortParts := strings.Split(hostAndPort, ":")
		if len(hostAndPortParts) < 2 {
			return fmt.Errorf("host-and-port (right) half doesn't have ':' symbol")
		}
		host = hostAndPortParts[0]
		rawPort := hostAndPortParts[1]

		var err error
		port, err = strconv.ParseInt(rawPort, 10, 32)
		if err != nil {
			return fmt.Errorf("port could not be parsed as int32 because %s", err)
		}

		return nil
	}()
	if err != nil {
		return "", "", "", 0, fmt.Errorf("failed to split %#+v into (username:password)@(host:port) because %s", target, err)
	}

	return username, password, host, port, nil
}

type Pool struct {
	Name    string   `yaml:"name"`
	Targets []string `yaml:"targets"`
}

func (p *Pool) GetName() string {
	return p.Name
}

func (p *Pool) Validate(_ ...string) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New("pool.name may not be empty")
	}

	for i, target := range p.Targets {
		_, _, _, _, err := ParseTarget(target)
		if err != nil {
			return fmt.Errorf("pool.targets[%d] is invalid because %s", i, err)
		}
	}

	return nil
}
