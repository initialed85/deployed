package types

import (
	"fmt"
	"log"
	"strings"

	"go.yaml.in/yaml/v4"
)

type Mapping struct {
	Spec  *Spec   `yaml:"spec"`
	Pools []*Pool `yaml:"pools"`
}

type Workspace struct {
	rootPath string    `yaml:"-"`
	Specs    []*Spec   `yaml:"specs,omitempty"`
	Pools    []*Pool   `yaml:"pools,omitempty"`
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

	err := Validate(w.Specs, rootPaths...)
	if err != nil {
		return err
	}

	downloadsByName := make(map[string][]Download)

	for i, spec := range w.Specs {
		for j, step := range spec.Steps {
			for _, download := range step.Downloads {
				if len(download.Name) == 0 {
					continue
				}

				downloads, ok := downloadsByName[download.Name]
				if !ok {
					downloads = make([]Download, 0)
				}

				downloads = append(downloads, download)

				downloadsByName[download.Name] = downloads
			}

			for k, upload := range step.Uploads {
				if !strings.HasPrefix(upload.Local, "?") {
					continue
				}

				uploadLocal := strings.TrimLeft(upload.Local, "?")

				downloads, ok := downloadsByName[uploadLocal]
				if !ok || len(downloads) == 0 {
					return fmt.Errorf("%T[%d].%T[%d] refers to %#+v which has no candidates at this point", spec, i, step, j, upload.Local)
				}

				if len(downloads) > 1 {
					return fmt.Errorf("%T[%d].%T[%d] refers to %#+v which has too many (%d) candidates at this point", spec, i, step, j, upload.Local, len(downloads))
				}

				upload.Local = downloads[0].Local

				step.Uploads[k] = upload
			}
		}
	}

	specByName, unnamedSpecs, err := GroupByName(w.Specs)
	if err != nil {
		return err
	}

	if len(unnamedSpecs) > 0 {
		return fmt.Errorf("found %d unnamed specs", len(unnamedSpecs))
	}

	//
	// pools
	//

	err = Validate(w.Pools, rootPaths...)
	if err != nil {
		return err
	}

	poolByName, unnamedPools, err := GroupByName(w.Pools)
	if err != nil {
		return err
	}

	if len(unnamedPools) > 0 {
		return fmt.Errorf("found %d unnamed pools", len(unnamedPools))
	}

	//
	// mappings
	//

	for i, mapping := range w.Mappings {
		spec, err := ResolveOrPassthru(mapping.Spec, specByName)
		if err != nil {
			return fmt.Errorf("%T.%s", spec, err)
		}

		mapping.Spec = spec

		for j, pool := range mapping.Pools {
			pool, err = ResolveOrPassthru(pool, poolByName)
			if err != nil {
				return fmt.Errorf("%T[%d].%s", pool, j, err)
			}

			mapping.Pools[j] = pool
		}

		w.Mappings[i] = mapping
	}

	return nil
}
