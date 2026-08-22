package types

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/initialed85/deployed/pkg/helpers/env"
	"go.yaml.in/yaml/v4"
)

type Mapping struct {
	SpecNames []string `yaml:"specs"`
	PoolNames []string `yaml:"pools"`
	Specs     []*Spec  `yaml:"-" json:"-"`
	Pools     []*Pool  `yaml:"-" json:"-"`
}

type Workspace struct {
	rootPath string    `yaml:"-" json:"-"`
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
		mapping.Specs = make([]*Spec, 0)

		for j, specName := range mapping.SpecNames {
			spec, err := Resolve(specName, specByName)
			if err != nil {
				return fmt.Errorf("mappings[%d].specs[%d]; %s", i, j, err)
			}

			mapping.Specs = append(mapping.Specs, spec)
		}

		mapping.Pools = make([]*Pool, 0)

		for j, poolName := range mapping.PoolNames {
			pool, err := Resolve(poolName, poolByName)
			if err != nil {
				return fmt.Errorf("mappings[%d].pools[%d]; %s", i, j, err)
			}

			mapping.Pools = append(mapping.Pools, pool)
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

func LoadWorkspace(workspaceYAMLPath string, requireSpecs bool, requirePools bool, requireTargets bool) (*Workspace, error) {
	workspaceYAMLPath, err := filepath.Abs(workspaceYAMLPath)
	if err != nil {
		return nil, err
	}

	rawWorkspace, err := os.ReadFile(workspaceYAMLPath)
	if err != nil {
		return nil, err
	}

	var workspace Workspace
	err = yaml.Load(rawWorkspace, &workspace, yaml.WithKnownFields())
	if err != nil {
		return nil, err
	}

	workspacePath, _ := filepath.Split(workspaceYAMLPath)

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	err = os.Chdir(workspacePath)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = os.Chdir(wd)
	}()

	err = workspace.Validate(workspacePath)
	if err != nil {
		return nil, err
	}

	if env.IsDebug {
		log.Printf("%s: ...\n\n%s\n", workspaceYAMLPath, workspace.PrettyFormat())
	}

	if len(workspace.Mappings) == 0 {
		return nil, fmt.Errorf("no mappings mappings specified")
	}

	for i, mapping := range workspace.Mappings {
		if requireSpecs {
			if len(mapping.Specs) == 0 {
				return nil, fmt.Errorf("mappings[%d] has no specs specified", i)
			}

			for j, spec := range mapping.Specs {
				if requireTargets {
					if len(spec.Steps) == 0 {
						return nil, fmt.Errorf("mappings[%d].pools[%d] has no targets specified", i, j)
					}
				}
			}
		}

		if requirePools {
			if len(mapping.Pools) == 0 {
				return nil, fmt.Errorf("mappings[%d] has no pools specified", i)
			}

			for j, pool := range mapping.Pools {
				if requireTargets {
					if len(pool.Targets) == 0 {
						return nil, fmt.Errorf("mappings[%d].pools[%d] has no targets specified", i, j)
					}
				}
			}
		}
	}

	return &workspace, nil
}
