package model

import "time"

type PropertyType string

const (
	TypeString  PropertyType = "string"
	TypeInteger PropertyType = "integer"
	TypeNumber  PropertyType = "number"
	TypeBoolean PropertyType = "boolean"
	TypeObject  PropertyType = "object"
	TypeArray   PropertyType = "array"
	TypeUnknown PropertyType = "unknown"
)

type Property struct {
	Name string `json:"name" yaml:"name"`

	Type PropertyType `json:"type" yaml:"type"`

	Nullable bool `json:"nullable" yaml:"nullable"`

	Required bool `json:"required" yaml:"required"`

	RequiredOnCreate bool `json:"requiredOnCreate" yaml:"requiredOnCreate"`

	ReadOnly bool `json:"readOnly" yaml:"readOnly"`

	Deprecated string `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	Enum []string `json:"enum,omitempty" yaml:"enum,omitempty"`

	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	Unit string `json:"unit,omitempty" yaml:"unit,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	Properties map[string]*Property `json:"properties,omitempty" yaml:"properties,omitempty"`

	Items *Property `json:"items,omitempty" yaml:"items,omitempty"`

	SpecRef string `json:"specRef,omitempty" yaml:"specRef,omitempty"`

	Unresolved bool `json:"unresolved,omitempty" yaml:"unresolved,omitempty"`
}

type Resource struct {
	Name string `json:"name" yaml:"name"`

	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	ODataType string `json:"odataType,omitempty" yaml:"odataType,omitempty"`

	Endpoints []string `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`

	Methods []string `json:"methods,omitempty" yaml:"methods,omitempty"`

	Properties map[string]*Property `json:"properties" yaml:"properties"`

	SpecRef string `json:"specRef,omitempty" yaml:"specRef,omitempty"`
}

type Spec struct {
	SourceVersion string `json:"sourceVersion,omitempty" yaml:"sourceVersion,omitempty"`

	Resources map[string]*Resource `json:"resources" yaml:"resources"`

	Sources []string `json:"sources,omitempty" yaml:"sources,omitempty"`

	GeneratedAt time.Time `json:"generatedAt" yaml:"generatedAt"`
}

func NewSpec() *Spec {
	return &Spec{
		Resources:   make(map[string]*Resource),
		GeneratedAt: time.Now().UTC(),
	}
}

func (s *Spec) Merge(other *Spec) {
	if other == nil {
		return
	}
	if s.Resources == nil {
		s.Resources = make(map[string]*Resource)
	}
	for name, incoming := range other.Resources {
		existing, ok := s.Resources[name]
		if !ok {
			s.Resources[name] = incoming
			continue
		}
		mergeResource(existing, incoming)
	}
	s.Sources = append(s.Sources, other.Sources...)
	if other.SourceVersion != "" {
		s.SourceVersion = other.SourceVersion
	}
}

func mergeResource(base, incoming *Resource) {
	if incoming.Version != "" {
		base.Version = incoming.Version
	}
	if incoming.ODataType != "" {
		base.ODataType = incoming.ODataType
	}
	if len(incoming.Endpoints) > 0 {
		base.Endpoints = incoming.Endpoints
	}
	if len(incoming.Methods) > 0 {
		base.Methods = incoming.Methods
	}
	if incoming.SpecRef != "" {
		base.SpecRef = incoming.SpecRef
	}
	if base.Properties == nil {
		base.Properties = make(map[string]*Property)
	}
	for name, prop := range incoming.Properties {
		base.Properties[name] = prop
	}
}
