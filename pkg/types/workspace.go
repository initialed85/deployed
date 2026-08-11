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

	err := ValidateMany(w.Specs, rootPaths...)
	if err != nil {
		return err
	}

	downloadsByName := make(map[string][]Download)
	downloadsByLocal := make(map[string][]Download)

	for i, spec := range w.Specs {
		for j, step := range spec.Steps {
			for _, download := range step.Downloads {
				if len(download.Name) == 0 {
					continue
				}

				downloadsForName, ok := downloadsByName[download.Name]
				if !ok {
					downloadsForName = make([]Download, 0)
				}

				downloadsForName = append(downloadsForName, download)

				downloadsByName[download.Name] = downloadsForName

				downloadsForLocal, ok := downloadsByLocal[download.Local]
				if !ok {
					downloadsForLocal = make([]Download, 0)
				}

				downloadsForLocal = append(downloadsForLocal, download)

				downloadsByLocal[download.Local] = downloadsForLocal
			}

			for k, upload := range step.Uploads {
				if !strings.HasPrefix(upload.Local, "?") {
					continue
				}

				uploadLocal := strings.TrimLeft(upload.Local, "?")

				downloads, ok := downloadsByName[uploadLocal]
				if !ok || len(downloads) == 0 {
					return fmt.Errorf(
						"specs[%d].steps[%d].uploads[%d] refers to download %#+v which has no candidates by the time its referenced (the deployment sequence up until this download being referenced has not caused this download to happen)",
						i, j, k, upload.Local,
					)
				}

				if len(downloads) > 1 {
					return fmt.Errorf(
						"specs[%d].steps[%d].uploads[%d]  refers to download %#+v which has too many (%d) candidates at this point (the deployment seqeuence up until this download being referenced has caused duplicate download names)",
						i, j, k, upload.Local, len(downloads),
					)
				}

				upload.Local = downloads[0].Local

				step.Uploads[k] = upload
			}
		}
	}

	downloadsByLocalErrorMsgs := make([]string, 0)

	for local, downloads := range downloadsByLocal {
		if len(downloads) <= 1 {
			continue
		}

		downloadsByLocalErrorMsgs = append(downloadsByLocalErrorMsgs, fmt.Sprintf("- %#+v is referred to %d times", local, len(downloads)))
	}

	if len(downloadsByLocalErrorMsgs) > 0 {
		return fmt.Errorf("one or more downloads have conflicting local file names (which would be overwritten throughout the rollout)\n\n%s", strings.Join(downloadsByLocalErrorMsgs, "\n"))
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

	err = ValidateMany(w.Pools, rootPaths...)
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
			return fmt.Errorf("mappings.spec.%s", err)
		}

		mapping.Spec = spec

		for j, pool := range mapping.Pools {
			pool, err = ResolveOrPassthru(pool, poolByName)
			if err != nil {
				return fmt.Errorf("mappings.pools[%d].%s", j, err)
			}

			mapping.Pools[j] = pool
		}

		w.Mappings[i] = mapping
	}

	for i, mapping := range w.Mappings {
		existingTargets := make(map[string]struct{})

		for j, pool := range mapping.Pools {
			adjustedTargets := make([]string, 0)

			for k, target := range pool.Targets {
				_, existing := existingTargets[target]
				if existing {
					log.Printf(
						"warning: mappings[%d].pools[%d].targets[%d] mentions target %s which has already been seen (i.e. it is a duplicate for this spec); it will be excluded",
						i, j, k, target,
					)
					continue
				}

				existingTargets[target] = struct{}{}

				adjustedTargets = append(adjustedTargets, target)
			}
		}
	}

	return nil
}
