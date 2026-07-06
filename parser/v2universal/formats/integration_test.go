package formats_test

import (
	"testing"

	"github.com/AliseMarfina/swordfish-verifier/parser/model"
	"github.com/AliseMarfina/swordfish-verifier/parser/v2universal"
	_ "github.com/AliseMarfina/swordfish-verifier/parser/v2universal/formats" // registers json/xml/yaml/pdf parsers
)

func TestUniversalParse_JSONOnly_MatchesV1(t *testing.T) {
	spec, err := v2universal.Parse(v2universal.Config{
		Sources: []v2universal.Source{
			{Path: "../../testdata/jsonschema"},
		},
		ResourceFilter: []string{"Volume"},
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	vol, ok := spec.Resources["Volume"]
	if !ok {
		t.Fatalf("expected Volume resource")
	}
	if vol.Version != "1.6.0" {
		t.Errorf("expected version from JSON schema, got %q", vol.Version)
	}
	if _, ok := vol.Properties["CapacityBytes"]; !ok {
		t.Errorf("expected CapacityBytes to be parsed from JSON Schema")
	}
}

func TestUniversalParse_XMLAddsRequiredOnCreateAndMethods(t *testing.T) {
	spec, err := v2universal.Parse(v2universal.Config{
		Sources: []v2universal.Source{
			{Path: "../../testdata/csdl/Volume_v1.xml"},
		},
		ResourceFilter: []string{"Volume"},
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	vol, ok := spec.Resources["Volume"]
	if !ok {
		t.Fatalf("expected Volume resource from CSDL")
	}
	if len(vol.Methods) == 0 {
		t.Errorf("expected HTTP methods to be derived from Capabilities annotations")
	}
	found := false
	for _, m := range vol.Methods {
		if m == "PATCH" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected PATCH in methods (Volume declares Capabilities.UpdateRestrictions Updatable=true), got %v", vol.Methods)
	}
	vt, ok := vol.Properties["VolumeType"]
	if !ok {
		t.Fatalf("expected VolumeType property from CSDL")
	}
	if vt.Deprecated == "" {
		t.Errorf("expected VolumeType deprecation note from Redfish.Deprecated annotation")
	}
	if len(vt.Enum) == 0 {
		t.Errorf("expected VolumeType enum values resolved from sibling EnumType declaration")
	}
}

func TestUniversalParse_PDFExtractsEndpoints(t *testing.T) {
	spec, err := v2universal.Parse(v2universal.Config{
		Sources: []v2universal.Source{
			{Path: "../../testdata/pdf/Swordfish_v1_2_8_Specification.pdf"},
		},
		ResourceFilter: []string{"Volume"},
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	vol, ok := spec.Resources["Volume"]
	if !ok {
		t.Fatalf("expected Volume resource from PDF heading extraction")
	}
	if len(vol.Endpoints) == 0 {
		t.Fatalf("expected at least one URI template extracted from the PDF 'URIs' subsection")
	}
	wantSuffix := "/Volumes/{VolumeId}"
	foundMatch := false
	for _, ep := range vol.Endpoints {
		if len(ep) >= len(wantSuffix) && ep[len(ep)-len(wantSuffix):] == wantSuffix {
			foundMatch = true
			break
		}
	}
	if !foundMatch {
		t.Errorf("expected an endpoint ending in %q, got %v", wantSuffix, vol.Endpoints)
	}
}

func TestUniversalParse_MergesAllFourSourcesWithOverrideWinning(t *testing.T) {
	spec, err := v2universal.Parse(v2universal.Config{
		Sources: []v2universal.Source{
			{Path: "../../testdata/jsonschema"},
			{Path: "../../testdata/csdl/Volume_v1.xml"},
			{Path: "../../testdata/pdf/Swordfish_v1_2_8_Specification.pdf"},
			{Path: "../../testdata/override/volume_override.yaml"},
		},
		ResourceFilter: []string{"Volume"},
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	vol, ok := spec.Resources["Volume"]
	if !ok {
		t.Fatalf("expected merged Volume resource")
	}

	if len(vol.Endpoints) == 0 {
		t.Errorf("expected endpoints to survive the merge (contributed by PDF, not overridden)")
	}
	if len(vol.Methods) == 0 {
		t.Errorf("expected methods contributed by XML to survive the merge (override didn't set its own subset of methods... wait it does)")
	}

	cap, ok := vol.Properties["CapacityBytes"]
	if !ok {
		t.Fatalf("expected CapacityBytes property to survive the merge")
	}
	if !cap.RequiredOnCreate {
		t.Errorf("expected the YAML override's requiredOnCreate:true to win for CapacityBytes")
	}
	if cap.SpecRef != "9.5.40.3 Volume properties, Table 148" {
		t.Errorf("expected the override's specRef to win, got %q", cap.SpecRef)
	}

	if vol.Version != "1.10.2" {
		t.Errorf("expected the override's version to win, got %q", vol.Version)
	}

	if spec.SourceVersion != "1.2.8" {
		t.Errorf("expected sourceVersion from override to be recorded, got %q", spec.SourceVersion)
	}

	_ = model.TypeInteger
}
