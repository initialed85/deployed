package types

import (
	"fmt"
	"os"
	"path/filepath"
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

type Mapping struct {
	Spec  Spec   `yaml:"spec"`
	Pools []Pool `yaml:"pools"`
}

type Workspace struct {
	rootPath string    `yaml:"-"`
	Specs    []Spec    `yaml:"specs"`
	Pools    []Pool    `yaml:"pools"`
	Mappings []Mapping `yaml:"mappings"`
}

func (w *Workspace) Validate(rootPath ...string) error {
	if len(rootPath) > 0 {
		w.rootPath = rootPath[0]
	}

	specByName := make(map[string]Spec)

	for i, spec := range w.Specs {
		spec.Name = strings.TrimSpace(spec.Name)
		if spec.Name == "" {
			return fmt.Errorf("spec[%d] has empty name", i)
		}

		_, existing := specByName[spec.Name]
		if existing {
			return fmt.Errorf("spec[%#+v] has name clash with existing spec", spec.Name)
		}

		for j, step := range spec.Steps {
			for k, upload := range step.Uploads {
				_, err := os.Stat(upload.Local)
				if err != nil {
					return fmt.Errorf("spec[%#+v].steps[%d].uploads[%d] refers to local file %#+v that does not exist", spec.Name, j, k, upload.Local)
				}
			}

			if step.Script != "" && strings.HasPrefix(step.Script, "@") {
				scriptPath := filepath.Join(w.rootPath, strings.TrimLeft(step.Script, "@"))

				stat, err := os.Stat(scriptPath)
				if err != nil {
					return fmt.Errorf("spec[%#+v].steps[%d].script refers to local file %#+v that does not exist", spec.Name, j, scriptPath)
				}

				if stat.IsDir() {
					return fmt.Errorf("spec[%#+v].steps[%d].script refers to local file %#+v that is a directory not a file", spec.Name, j, scriptPath)
				}

				rawScript, err := os.ReadFile(scriptPath)
				if err != nil {
					return fmt.Errorf("spec[%#+v].steps[%d].script refers to local file %#+v that could not be read because %s", spec.Name, j, scriptPath, err)
				}

				step.Script = string(rawScript)
			}

			spec.Steps[j] = step
		}

		specByName[spec.Name] = spec
	}

	poolByName := make(map[string]Pool)

	for i, pool := range w.Pools {
		pool.Name = strings.TrimSpace(pool.Name)
		if pool.Name == "" {
			return fmt.Errorf("pool[%d] has empty name", i)
		}

		_, existing := poolByName[pool.Name]
		if existing {
			return fmt.Errorf("pool[%#+v] has name clash with existing pool", pool.Name)
		}

		for j, target := range pool.Targets {
			_, _, _, _, err := ParseTarget(target)
			if err != nil {
				return fmt.Errorf("pool[%#+v].target[%d] is invalid because %s", pool.Name, j, err)
			}
		}

		poolByName[pool.Name] = pool
	}

	for i, mapping := range w.Mappings {
		if strings.HasPrefix(mapping.Spec.Name, "@") {
			specName := strings.TrimLeft(mapping.Spec.Name, "@")

			spec, ok := specByName[specName]
			if !ok {
				return fmt.Errorf("mapping[%d].spec.name %#+v is not in list of known specs", i, specName)
			}

			mapping.Spec = spec
		} else {
			panic("TODO")
		}

		for j, pool := range mapping.Pools {
			if strings.HasPrefix(pool.Name, "@") {
				specName := strings.TrimLeft(pool.Name, "@")

				spec, ok := poolByName[specName]
				if !ok {
					return fmt.Errorf("mapping[%d].pool[%d].name %#+v is not in list of known specs", i, j, specName)
				}

				pool = spec
			} else {
				panic("TODO")
			}
		}
	}

	return nil
}
