package formats

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AliseMarfina/swordfish-verifier/parser/model"
	"github.com/AliseMarfina/swordfish-verifier/parser/v2universal"
)

type YAMLFormat struct{}

func (YAMLFormat) Name() string { return "yaml" }

func (YAMLFormat) Supports(path string) bool {
	ok, _ := dirOrFileHasSuffix(path, ".yaml")
	if ok {
		return true
	}
	ok, _ = dirOrFileHasSuffix(path, ".yml")
	return ok
}

type yamlDoc struct {
	SourceVersion string                     `yaml:"sourceVersion"`
	Resources     map[string]*model.Resource `yaml:"resources"`
}

func (YAMLFormat) Parse(path string, resourceFilter []string) (*model.Spec, error) {
	files, err := listOverrideFiles(path)
	if err != nil {
		return nil, err
	}

	filter := make(map[string]bool, len(resourceFilter))
	for _, f := range resourceFilter {
		filter[f] = true
	}

	spec := model.NewSpec()
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("yaml: reading %s: %w", file, err)
		}
		var doc yamlDoc
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("yaml: invalid override file %s: %w", file, err)
		}
		if doc.SourceVersion != "" {
			spec.SourceVersion = doc.SourceVersion
		}
		spec.Sources = append(spec.Sources, file)

		for name, res := range doc.Resources {
			if len(filter) > 0 && !filter[name] {
				continue
			}
			res.Name = name
			if res.SpecRef == "" {
				res.SpecRef = "override:" + file
			}
			for propName, prop := range res.Properties {
				prop.Name = propName
				if prop.SpecRef == "" {
					prop.SpecRef = res.SpecRef
				}
			}
			spec.Resources[name] = res
		}
	}
	return spec, nil
}

func listOverrideFiles(path string) ([]string, error) {
	files, err := listFilesLocal(path, ".yaml", ".yml")
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("yaml: no .yaml/.yml files found under %s", path)
	}
	return files, nil
}

func listFilesLocal(path string, exts ...string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		for _, ext := range exts {
			if strings.HasSuffix(lower, ext) {
				out = append(out, path+"/"+e.Name())
				break
			}
		}
	}
	return out, nil
}

func init() {
	v2universal.Register(YAMLFormat{})
}
