package types

import (
	"fmt"
	"log"
	"strings"

	"go.yaml.in/yaml/v4"
)

type Mapping struct {
	Spec  Spec   `yaml:"spec"`
	Pools []Pool `yaml:"pools"`
}

type Workspace struct {
	rootPath string    `yaml:"-"`
	Specs    []Spec    `yaml:"specs,omitempty"`
	Pools    []Pool    `yaml:"pools,omitempty"`
	Mappings []Mapping `yaml:"mappings"`
}

func (w *Workspace) PrettyFormat() string {
	t := Workspace{
		Specs:    nil,
		Pools:    nil,
		Mappings: w.Mappings,
	}

	b, err := yaml.Marshal(t)
	if err != nil {
		log.Printf("warning: failed to marshal %#+v because %s; .PrettyFormat() will return an empty string", t, err)
	}

	return string(b)
}

func (w *Workspace) Validate(rootPaths ...string) error {
	if len(rootPaths) > 0 {
		w.rootPath = rootPaths[0]
	}

	//
	// specs
	//

	specByName := make(map[string]Spec)

	for i, spec := range w.Specs {
		err := spec.Validate(w.rootPath)
		if err != nil {
			return fmt.Errorf("spec[%d]%s", i, err)
		}

		_, existing := specByName[spec.Name]
		if existing {
			return fmt.Errorf("spec[%d] has name clash on %#+v with existing spec", i, spec.Name)
		}

		specByName[spec.Name] = spec
	}

	//
	// pools
	//

	poolByName := make(map[string]Pool)

	for i, pool := range w.Pools {
		err := pool.Validate()
		if err != nil {
			return fmt.Errorf("pool[%d]%s", i, err)
		}

		_, existing := poolByName[pool.Name]
		if existing {
			return fmt.Errorf("pool[%#+v] has name clash with existing pool", pool.Name)
		}

		poolByName[pool.Name] = pool
	}

	//
	// mappings
	//

	for i, mapping := range w.Mappings {
		if strings.HasPrefix(mapping.Spec.Name, "@") {
			specName := strings.TrimLeft(mapping.Spec.Name, "@")

			spec, ok := specByName[specName]
			if !ok {
				return fmt.Errorf("mapping[%d].spec.name %#+v is not in list of known specs", i, specName)
			}

			mapping.Spec = spec
		} else {
			err := mapping.Spec.Validate(w.rootPath)
			if err != nil {
				return fmt.Errorf("mapping[%d]%s", i, err)
			}
		}

		for j, pool := range mapping.Pools {
			if strings.HasPrefix(pool.Name, "@") {
				poolName := strings.TrimLeft(pool.Name, "@")

				pool, ok := poolByName[poolName]
				if !ok {
					return fmt.Errorf("mapping[%d].pool[%d].name %#+v is not in list of known pools", i, j, poolName)
				}

				mapping.Pools[j] = pool
			} else {
				err := pool.Validate()
				if err != nil {
					return fmt.Errorf("mapping[%d]%s", i, err)
				}
			}
		}

		w.Mappings[i] = mapping
	}

	return nil
}
