package v1json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/AliseMarfina/swordfish-verifier/internal/model"
)

type Config struct {
	SchemaDir string

	ResourceFilter []string

	SpecVersion string
}

type rawSchema struct {
	ID          string                     `json:"$id"`
	Ref         string                     `json:"$ref"`
	Title       string                     `json:"title"`
	Definitions map[string]json.RawMessage `json:"definitions"`
}

func Parse(cfg Config) (*model.Spec, error) {
	if cfg.SchemaDir == "" {
		return nil, fmt.Errorf("v1json: SchemaDir is required")
	}
	entries, err := os.ReadDir(cfg.SchemaDir)
	if err != nil {
		return nil, fmt.Errorf("v1json: reading schema dir: %w", err)
	}

	filter := toSet(cfg.ResourceFilter)

	docs := make(map[string]*loadedDoc)
	var sources []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		full := filepath.Join(cfg.SchemaDir, e.Name())
		data, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("v1json: reading %s: %w", e.Name(), err)
		}
		var raw rawSchema
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("v1json: invalid JSON schema in %s: %w", e.Name(), err)
		}
		key := strings.TrimSuffix(e.Name(), ".json")
		docs[key] = &loadedDoc{
			fileName: e.Name(),
			schema:   raw,
		}
		sources = append(sources, full)
	}

	resolver := &refResolver{docs: docs}

	spec := model.NewSpec()
	spec.SourceVersion = cfg.SpecVersion
	spec.Sources = sources

	for key, doc := range docs {
		resourceName, version := splitVersionedName(key)
		if resourceName == "" {
			continue
		}
		if len(filter) > 0 && !filter[resourceName] {
			continue
		}
		defRaw, ok := doc.schema.Definitions[resourceName]
		if !ok {
			continue
		}

		var def objectDef
		if err := json.Unmarshal(defRaw, &def); err != nil {
			return nil, fmt.Errorf("v1json: parsing definition %s in %s: %w", resourceName, doc.fileName, err)
		}

		props := make(map[string]*model.Property, len(def.Properties))
		for name, propRaw := range def.Properties {
			if isODataAnnotationProperty(name) {
				continue
			}
			prop, err := resolver.resolveProperty(propRaw, doc, resourceName+"."+name)
			if err != nil {
				return nil, fmt.Errorf("v1json: property %s.%s: %w", resourceName, name, err)
			}
			prop.Name = name
			prop.Required = contains(def.Required, name)
			props[name] = prop
		}

		resource := &model.Resource{
			Name:       resourceName,
			Version:    version,
			ODataType:  doc.schema.Title,
			Properties: props,
			SpecRef:    doc.schema.ID,
		}

		if existing, ok := spec.Resources[resourceName]; !ok || version != "" && existing.Version == "" {
			spec.Resources[resourceName] = resource
		}
	}

	return spec, nil
}

type loadedDoc struct {
	fileName string
	schema   rawSchema
}

type objectDef struct {
	Type       interface{}                `json:"type"`
	Required   []string                   `json:"required"`
	Properties map[string]json.RawMessage `json:"properties"`
}

type propertySchema struct {
	Type        interface{}                `json:"type"`
	Ref         string                     `json:"$ref"`
	AnyOf       []propertySchema           `json:"anyOf"`
	Enum        []string                   `json:"enum"`
	ReadOnly    bool                       `json:"readonly"`
	Deprecated  interface{}                `json:"deprecated"`
	Description string                     `json:"description"`
	Units       string                     `json:"units"`
	Items       json.RawMessage            `json:"items"`
	Properties  map[string]json.RawMessage `json:"properties"`
}

type refResolver struct {
	docs map[string]*loadedDoc
}

func (r *refResolver) resolveProperty(raw json.RawMessage, doc *loadedDoc, ref string) (*model.Property, error) {
	var ps propertySchema
	if err := json.Unmarshal(raw, &ps); err != nil {
		return nil, err
	}
	return r.fromSchema(ps, doc, ref, 0)
}

func (r *refResolver) fromSchema(ps propertySchema, doc *loadedDoc, ref string, depth int) (*model.Property, error) {
	p := &model.Property{SpecRef: ref, Description: ps.Description, Unit: ps.Units, ReadOnly: ps.ReadOnly}
	if s, ok := ps.Deprecated.(string); ok {
		p.Deprecated = s
	} else if b, ok := ps.Deprecated.(bool); ok && b {
		p.Deprecated = "deprecated"
	}

	if len(ps.AnyOf) > 0 {
		p.Nullable = anyOfHasNull(ps.AnyOf)
		for _, alt := range ps.AnyOf {
			if alt.Ref != "" {
				resolved, err := r.followRef(alt.Ref, doc, ref, depth)
				if err != nil {
					return nil, err
				}
				resolved.Nullable = p.Nullable || resolved.Nullable
				resolved.Description = firstNonEmpty(p.Description, resolved.Description)
				resolved.ReadOnly = p.ReadOnly || resolved.ReadOnly
				resolved.Unit = firstNonEmpty(p.Unit, resolved.Unit)
				resolved.Deprecated = firstNonEmpty(p.Deprecated, resolved.Deprecated)
				resolved.SpecRef = ref
				return resolved, nil
			}
		}
	}

	if ps.Ref != "" && depth < 6 {
		resolved, err := r.followRef(ps.Ref, doc, ref, depth)
		if err != nil {
			return nil, err
		}
		resolved.Description = firstNonEmpty(p.Description, resolved.Description)
		resolved.ReadOnly = p.ReadOnly || resolved.ReadOnly
		resolved.Unit = firstNonEmpty(p.Unit, resolved.Unit)
		resolved.Deprecated = firstNonEmpty(p.Deprecated, resolved.Deprecated)
		resolved.SpecRef = ref
		return resolved, nil
	}

	types, nullable := normalizeTypes(ps.Type)
	p.Nullable = p.Nullable || nullable
	p.Type = primaryType(types)
	p.Enum = ps.Enum

	switch p.Type {
	case model.TypeArray:
		if len(ps.Items) > 0 {
			var itemSchema propertySchema
			if err := json.Unmarshal(ps.Items, &itemSchema); err != nil {
				return nil, err
			}
			item, err := r.fromSchema(itemSchema, doc, ref+"[]", depth+1)
			if err != nil {
				return nil, err
			}
			p.Items = item
		}
	case model.TypeObject:
		if len(ps.Properties) > 0 {
			p.Properties = make(map[string]*model.Property, len(ps.Properties))
			for name, raw := range ps.Properties {
				if isODataAnnotationProperty(name) {
					continue
				}
				var nested propertySchema
				if err := json.Unmarshal(raw, &nested); err != nil {
					return nil, err
				}
				child, err := r.fromSchema(nested, doc, ref+"."+name, depth+1)
				if err != nil {
					return nil, err
				}
				child.Name = name
				p.Properties[name] = child
			}
		}
	}

	if p.Type == "" {
		p.Type = model.TypeUnknown
	}
	return p, nil
}

func (r *refResolver) followRef(refStr string, from *loadedDoc, callSiteRef string, depth int) (*model.Property, error) {
	fileKey, pointer := splitRef(refStr)
	target := from
	if fileKey != "" {
		key := strings.TrimSuffix(fileKey, ".json")
		var ok bool
		target, ok = r.docs[key]
		if !ok {
			return &model.Property{
				Type:       model.TypeUnknown,
				SpecRef:    refStr,
				Unresolved: true,
			}, nil
		}
	}

	defName := lastPathSegment(pointer)
	defRaw, ok := target.schema.Definitions[defName]
	if !ok {
		return &model.Property{Type: model.TypeUnknown, SpecRef: refStr, Unresolved: true}, nil
	}
	var nested propertySchema
	if err := json.Unmarshal(defRaw, &nested); err != nil {
		return nil, err
	}
	return r.fromSchema(nested, target, callSiteRef, depth+1)
}

var versionedNameRe = regexp.MustCompile(`^([A-Za-z0-9]+)\.v(\d+)_(\d+)_(\d+)$`)

func splitVersionedName(key string) (name, version string) {
	if m := versionedNameRe.FindStringSubmatch(key); m != nil {
		return m[1], fmt.Sprintf("%s.%s.%s", m[2], m[3], m[4])
	}
	return key, ""
}

func splitRef(ref string) (fileKey, pointer string) {
	hashIdx := strings.Index(ref, "#")
	pointer = ref
	base := ref
	if hashIdx >= 0 {
		base = ref[:hashIdx]
		pointer = ref[hashIdx+1:]
	}
	if base == "" {
		return "", pointer
	}
	return filepath.Base(base), pointer
}

func lastPathSegment(pointer string) string {
	parts := strings.Split(strings.Trim(pointer, "/"), "/")
	return parts[len(parts)-1]
}

func normalizeTypes(t interface{}) (types []string, nullable bool) {
	switch v := t.(type) {
	case string:
		types = []string{v}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				types = append(types, s)
			}
		}
	}
	filtered := types[:0]
	for _, tp := range types {
		if tp == "null" {
			nullable = true
			continue
		}
		filtered = append(filtered, tp)
	}
	return filtered, nullable
}

func primaryType(types []string) model.PropertyType {
	if len(types) == 0 {
		return ""
	}
	switch types[0] {
	case "string":
		return model.TypeString
	case "integer":
		return model.TypeInteger
	case "number":
		return model.TypeNumber
	case "boolean":
		return model.TypeBoolean
	case "object":
		return model.TypeObject
	case "array":
		return model.TypeArray
	default:
		return model.TypeUnknown
	}
}

func anyOfHasNull(alts []propertySchema) bool {
	for _, a := range alts {
		if s, ok := a.Type.(string); ok && s == "null" {
			return true
		}
	}
	return false
}

func isODataAnnotationProperty(name string) bool {
	return strings.Contains(name, "@odata") || strings.Contains(name, "@Redfish") || strings.Contains(name, "@Message")
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func toSet(list []string) map[string]bool {
	set := make(map[string]bool, len(list))
	for _, v := range list {
		set[v] = true
	}
	return set
}

func SortedResourceNames(spec *model.Spec) []string {
	names := make([]string, 0, len(spec.Resources))
	for name := range spec.Resources {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
