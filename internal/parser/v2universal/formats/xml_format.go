package formats

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/AliseMarfina/swordfish-verifier/internal/model"
	"github.com/AliseMarfina/swordfish-verifier/internal/parser/v2universal"
)

type XMLFormat struct{}

func (XMLFormat) Name() string { return "xml" }

func (XMLFormat) Supports(path string) bool {
	ok, _ := dirOrFileHasSuffix(path, ".xml")
	return ok
}

type edmx struct {
	XMLName xml.Name `xml:"Edmx"`
	Schemas []schema `xml:"DataServices>Schema"`
}

type schema struct {
	Namespace  string       `xml:"Namespace,attr"`
	EntityType []entityType `xml:"EntityType"`
	EnumType   []enumType   `xml:"EnumType"`
}

type entityType struct {
	Name       string        `xml:"Name,attr"`
	BaseType   string        `xml:"BaseType,attr"`
	Abstract   string        `xml:"Abstract,attr"`
	Property   []xmlProperty `xml:"Property"`
	Annotation []annotation  `xml:"Annotation"`
}

type xmlProperty struct {
	Name       string       `xml:"Name,attr"`
	Type       string       `xml:"Type,attr"`
	Nullable   string       `xml:"Nullable,attr"`
	Annotation []annotation `xml:"Annotation"`
}

type annotation struct {
	Term       string `xml:"Term,attr"`
	String     string `xml:"String,attr"`
	Bool       string `xml:"Bool,attr"`
	EnumMember string `xml:"EnumMember,attr"`
	Record     struct {
		PropertyValue []struct {
			Property string `xml:"Property,attr"`
			Bool     string `xml:"Bool,attr"`
		} `xml:"PropertyValue"`
	} `xml:"Record"`
}

type enumType struct {
	Name   string       `xml:"Name,attr"`
	Member []enumMember `xml:"Member"`
}

type enumMember struct {
	Name string `xml:"Name,attr"`
}

func (f XMLFormat) Parse(path string, resourceFilter []string) (*model.Spec, error) {
	files, err := listFilesLocal(path, ".xml")
	if err != nil {
		return nil, err
	}
	filter := make(map[string]bool, len(resourceFilter))
	for _, r := range resourceFilter {
		filter[r] = true
	}

	spec := model.NewSpec()
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("xml: reading %s: %w", file, err)
		}
		var doc edmx
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("xml: invalid CSDL document %s: %w", file, err)
		}
		spec.Sources = append(spec.Sources, file)

		enums := collectEnums(doc.Schemas)
		abstractAnnotations := collectAbstractAnnotations(doc.Schemas)

		for _, sch := range doc.Schemas {
			for _, et := range sch.EntityType {
				if et.Abstract == "true" {
					continue
				}
				if len(filter) > 0 && !filter[et.Name] {
					continue
				}
				version := versionFromNamespace(sch.Namespace)
				resource := spec.Resources[et.Name]
				if resource == nil {
					resource = &model.Resource{Name: et.Name, Properties: map[string]*model.Property{}}
					spec.Resources[et.Name] = resource
				}
				if version != "" {
					resource.Version = version
				}
				resource.SpecRef = fmt.Sprintf("%s#Schema[%s]/EntityType[%s]", file, sch.Namespace, et.Name)
				applyMethodCapabilities(resource, abstractAnnotations[et.Name])
				applyMethodCapabilities(resource, et.Annotation)

				for _, p := range et.Property {
					prop := propertyFromXML(p, enums)
					prop.SpecRef = resource.SpecRef + "/Property[" + p.Name + "]"
					resource.Properties[p.Name] = prop
				}
			}
		}
	}
	return spec, nil
}

func collectEnums(schemas []schema) map[string][]string {
	enums := make(map[string][]string)
	for _, sch := range schemas {
		for _, e := range sch.EnumType {
			values := make([]string, 0, len(e.Member))
			for _, m := range e.Member {
				values = append(values, m.Name)
			}
			enums[e.Name] = values
		}
	}
	return enums
}

func propertyFromXML(p xmlProperty, enums map[string][]string) *model.Property {
	prop := &model.Property{
		Name:     p.Name,
		Nullable: p.Nullable != "false",
	}
	prop.Type, prop.Enum = mapEdmType(p.Type, enums)

	for _, a := range p.Annotation {
		switch a.Term {
		case "OData.Description":
			prop.Description = a.String
		case "OData.LongDescription":
			if prop.Description == "" {
				prop.Description = a.String
			}
		case "Measures.Unit":
			prop.Unit = a.String
		case "Redfish.Deprecated":
			prop.Deprecated = a.String
			if prop.Deprecated == "" {
				prop.Deprecated = "deprecated"
			}
		case "Redfish.RequiredOnCreate":
			prop.RequiredOnCreate = true
		case "Redfish.Required":
			prop.Required = true
		case "OData.Permissions":
			prop.ReadOnly = a.EnumMember == "OData.Permission/Read"
		}
	}
	return prop
}

func mapEdmType(edmType string, enums map[string][]string) (model.PropertyType, []string) {
	t := strings.TrimPrefix(edmType, "Collection(")
	t = strings.TrimSuffix(t, ")")
	switch t {
	case "Edm.String":
		return model.TypeString, nil
	case "Edm.Int16", "Edm.Int32", "Edm.Int64":
		return model.TypeInteger, nil
	case "Edm.Decimal", "Edm.Double":
		return model.TypeNumber, nil
	case "Edm.Boolean":
		return model.TypeBoolean, nil
	case "Edm.Guid":
		return model.TypeString, nil
	}
	simple := t
	if idx := strings.LastIndex(t, "."); idx >= 0 {
		simple = t[idx+1:]
	}
	if values, ok := enums[simple]; ok {
		return model.TypeString, values
	}
	if strings.HasPrefix(edmType, "Collection(") {
		return model.TypeArray, nil
	}
	return model.TypeObject, nil
}

func versionFromNamespace(namespace string) string {
	parts := strings.SplitN(namespace, ".v", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.ReplaceAll(parts[1], "_", ".")
}

func applyMethodCapabilities(resource *model.Resource, annotations []annotation) {
	methods := map[string]bool{"GET": true}
	for _, m := range resource.Methods {
		methods[m] = true
	}
	for _, a := range annotations {
		switch a.Term {
		case "Capabilities.InsertRestrictions":
			if boolProp(a, "Insertable") {
				methods["POST"] = true
			}
		case "Capabilities.UpdateRestrictions":
			if boolProp(a, "Updatable") {
				methods["PATCH"] = true
				methods["PUT"] = true
			}
		case "Capabilities.DeleteRestrictions":
			if boolProp(a, "Deletable") {
				methods["DELETE"] = true
			}
		}
	}
	resource.Methods = resource.Methods[:0]
	for _, m := range []string{"GET", "POST", "PATCH", "PUT", "DELETE"} {
		if methods[m] {
			resource.Methods = append(resource.Methods, m)
		}
	}
}

func collectAbstractAnnotations(schemas []schema) map[string][]annotation {
	out := make(map[string][]annotation)
	for _, sch := range schemas {
		for _, et := range sch.EntityType {
			if et.Abstract == "true" {
				out[et.Name] = et.Annotation
			}
		}
	}
	return out
}

func boolProp(a annotation, propName string) bool {
	for _, pv := range a.Record.PropertyValue {
		if pv.Property == propName {
			return pv.Bool == "true"
		}
	}
	return false
}

func init() {
	v2universal.Register(XMLFormat{})
}
