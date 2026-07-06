package v1json

import (
	"testing"

	"github.com/AliseMarfina/swordfish-verifier/parser/model"
)

func TestParse_Volume(t *testing.T) {
	spec, err := Parse(Config{
		SchemaDir:      "../testdata/jsonschema",
		ResourceFilter: []string{"Volume", "StoragePool"},
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	vol, ok := spec.Resources["Volume"]
	if !ok {
		t.Fatalf("expected Volume resource to be parsed, got: %v", SortedResourceNames(spec))
	}
	if vol.Version != "1.6.0" {
		t.Errorf("expected version 1.6.0, got %q", vol.Version)
	}
	if vol.ODataType != "#Volume.v1_6_0.Volume" {
		t.Errorf("unexpected odata type: %q", vol.ODataType)
	}

	cap, ok := vol.Properties["CapacityBytes"]
	if !ok {
		t.Fatalf("expected CapacityBytes property")
	}
	if cap.Type != model.TypeInteger {
		t.Errorf("expected CapacityBytes type integer, got %s", cap.Type)
	}
	if !cap.Nullable {
		t.Errorf("expected CapacityBytes to be nullable")
	}
	if cap.Unit != "By" {
		t.Errorf("expected unit 'By', got %q", cap.Unit)
	}

	id, ok := vol.Properties["Id"]
	if !ok {
		t.Fatalf("expected Id property (resolved via cross-file $ref to Resource.json)")
	}
	if id.Type != model.TypeString {
		t.Errorf("expected Id type string (resolved from Resource.json), got %s (unresolved=%v)", id.Type, id.Unresolved)
	}
	if !id.Required {
		t.Errorf("expected Id to be required (listed in Volume's top-level required[])")
	}

	vt, ok := vol.Properties["VolumeType"]
	if !ok {
		t.Fatalf("expected VolumeType property")
	}
	if vt.Deprecated == "" {
		t.Errorf("expected VolumeType to carry a deprecation note")
	}
	if len(vt.Enum) == 0 {
		t.Errorf("expected VolumeType enum values to be resolved from Volume.json, got none (unresolved=%v)", vt.Unresolved)
	}

	if _, ok := spec.Resources["StoragePool"]; !ok {
		t.Errorf("expected StoragePool resource to be parsed too")
	}
}

func TestParse_ResourceFilterExcludesUnwanted(t *testing.T) {
	spec, err := Parse(Config{
		SchemaDir:      "../testdata/jsonschema",
		ResourceFilter: []string{"Volume"},
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if _, ok := spec.Resources["StoragePool"]; ok {
		t.Errorf("expected StoragePool to be excluded by ResourceFilter")
	}
}
