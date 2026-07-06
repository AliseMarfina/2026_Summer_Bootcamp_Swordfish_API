package formats

import (
	"github.com/AliseMarfina/swordfish-verifier/parser/model"
	"github.com/AliseMarfina/swordfish-verifier/parser/v1json"
	"github.com/AliseMarfina/swordfish-verifier/parser/v2universal"
)

type JSONFormat struct{}

func (JSONFormat) Name() string { return "json" }

func (JSONFormat) Supports(path string) bool {
	matches, _ := hasJSONSchemaFiles(path)
	return matches
}

func (JSONFormat) Parse(path string, resourceFilter []string) (*model.Spec, error) {
	dir, err := schemaDirFor(path)
	if err != nil {
		return nil, err
	}
	return v1json.Parse(v1json.Config{
		SchemaDir:      dir,
		ResourceFilter: resourceFilter,
	})
}

func hasJSONSchemaFiles(path string) (bool, error) {
	return dirOrFileHasSuffix(path, ".json")
}

func init() {
	v2universal.Register(JSONFormat{})
}
