package comparator_test

import (
	"testing"

	"github.com/AliseMarfina/swordfish-verifier/internal/comparator"
	"github.com/AliseMarfina/swordfish-verifier/parser/model"
)

func TestCompare_Pass(t *testing.T) {
	spec := &model.Spec{
		Resources: map[string]*model.Resource{
			"TestResource": {
				Properties: map[string]*model.Property{
					"Id":   {Name: "Id", Type: model.TypeString, Required: true},
					"Name": {Name: "Name", Type: model.TypeString, Required: false},
				},
			},
		},
	}
	jsonData := []byte(`{"Id": "123", "Name": "test"}`)
	results, err := comparator.Compare("TestResource", spec, jsonData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != "PASS" {
			t.Errorf("Expected PASS for field %s, got %s", r.Field, r.Status)
		}
	}
}

func TestCompare_FailMissingRequired(t *testing.T) {
	spec := &model.Spec{
		Resources: map[string]*model.Resource{
			"TestResource": {
				Properties: map[string]*model.Property{
					"Id": {Name: "Id", Type: model.TypeString, Required: true},
				},
			},
		},
	}
	jsonData := []byte(`{}`) //Id отсутствует
	results, err := comparator.Compare("TestResource", spec, jsonData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Status != "FAIL" {
		t.Errorf("Expected FAIL, got %s", results[0].Status)
	}
	if results[0].Field != "Id" {
		t.Errorf("Expected field 'Id', got %s", results[0].Field)
	}
}

func TestCompare_NotSupported(t *testing.T) {
	spec := &model.Spec{Resources: map[string]*model.Resource{}}
	jsonData := []byte(`{}`)
	results, err := comparator.Compare("NonExistent", spec, jsonData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Status != "NOT_SUPPORTED" {
		t.Errorf("Expected NOT_SUPPORTED, got %s", results[0].Status)
	}
}