package comparator

import (
	"encoding/json"
	"fmt"

	// "reflect"
	// "strings"
	"github.com/AliseMarfina/swordfish-verifier/internal/model"
)

type CheckResult struct {
	Resource string `json:"resource"`
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Status   string `json:"status"` // PASS, FAIL, NOT_SUPPORTED
	Message  string `json:"message,omitempty"`
}

func Compare(resourceName string, spec *model.Spec, actualJSON []byte) ([]CheckResult, error) {
	resource, ok := spec.Resources[resourceName]
	if !ok {
		return []CheckResult{{
			Resource: resourceName,
			Status:   "NOT_SUPPORTED",
			Message:  "Resource not found in specification",
		}}, nil
	}

	var actual map[string]interface{}
	if err := json.Unmarshal(actualJSON, &actual); err != nil {
		return nil, fmt.Errorf("failed to parse actual JSON: %w", err)
	}
	var results []CheckResult
	// Проверяем каждое свойство, описанное в спецификации
	for fieldName, prop := range resource.Properties {
		path := fieldName
		result := checkProperty(prop, actual, path, resourceName)
		if result != nil {
			results = append(results, *result)
		}
	}
	return results, nil
}

func checkProperty(prop *model.Property, data map[string]interface{}, fieldPath, resourceName string) *CheckResult {
	val, exists := data[prop.Name]

	// Если поле обязательно и отсутствует -> FAIL
	if !exists {
		if prop.Required {
			return &CheckResult{
				Resource: resourceName,
				Field:    fieldPath,
				Expected: fmt.Sprintf("required field of type %s", prop.Type),
				Actual:   "missing",
				Status:   "FAIL",
				Message:  "Required field is missing",
			}
		}
		return nil
	}

	actualType := getJSONType(val)
	expectedType := string(prop.Type)
	if len(prop.Enum) > 0 {
		strVal, ok := val.(string)
		if !ok {
			return &CheckResult{
				Resource: resourceName,
				Field:    fieldPath,
				Expected: fmt.Sprintf("enum value, type string, one of %v", prop.Enum),
				Actual:   fmt.Sprintf("%v (%T)", val, val),
				Status:   "FAIL",
				Message:  "Value is not a string, expected enum",
			}
		}
		found := false
		for _, enumVal := range prop.Enum {
			if strVal == enumVal {
				found = true
				break
			}
		}
		if !found {
			return &CheckResult{
				Resource: resourceName,
				Field:    fieldPath,
				Expected: fmt.Sprintf("enum value one of %v", prop.Enum),
				Actual:   strVal,
				Status:   "FAIL",
				Message:  "Value not in enum list",
			}
		}
		if prop.Type == model.TypeString {
			return &CheckResult{
				Resource: resourceName,
				Field:    fieldPath,
				Expected: fmt.Sprintf("type %s, enum valid", prop.Type),
				Actual:   strVal,
				Status:   "PASS",
			}
		}
	}

	if expectedType != string(model.TypeUnknown) && expectedType != actualType {
		return &CheckResult{
			Resource: resourceName,
			Field:    fieldPath,
			Expected: fmt.Sprintf("type %s", expectedType),
			Actual:   fmt.Sprintf("%v (%s)", val, actualType),
			Status:   "FAIL",
			Message:  fmt.Sprintf("Type mismatch: expected %s, got %s", expectedType, actualType),
		}
	}

	if prop.Type == model.TypeObject && prop.Properties != nil && len(prop.Properties) > 0 {
		objData, ok := val.(map[string]interface{})
		if !ok {
			return &CheckResult{
				Resource: resourceName,
				Field:    fieldPath,
				Expected: "object",
				Actual:   fmt.Sprintf("%v (%T)", val, val),
				Status:   "FAIL",
				Message:  "Expected object, got non-object",
			}
		}

		for nestedName, nestedProp := range prop.Properties {
			nestedPath := fieldPath + "." + nestedName
			nestedResult := checkProperty(nestedProp, objData, nestedPath, resourceName)
			if nestedResult != nil {
			}
		}
	}

	// Если все проверки пройдены – PASS
	return &CheckResult{
		Resource: resourceName,
		Field:    fieldPath,
		Expected: fmt.Sprintf("type %s", expectedType),
		Actual:   fmt.Sprintf("%v", val),
		Status:   "PASS",
	}
}

// getJSONType возвращает строковое представление типа JSON-значения
func getJSONType(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case float64, float32, int, int64, int32, int16, int8, uint64, uint32, uint16, uint8:
		return "number"
	case bool:
		return "boolean"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		return "unknown"
	}
}

func checkObjectProperties(prop *model.Property, data map[string]interface{}, prefix string, resourceName string) []CheckResult {
	var results []CheckResult
	if prop.Type == model.TypeObject && prop.Properties != nil {
		for nestedName, nestedProp := range prop.Properties {
			path := prefix + "." + nestedName
			val, exists := data[nestedName]
			if !exists {
				if nestedProp.Required {
					results = append(results, CheckResult{
						Resource: resourceName,
						Field:    path,
						Expected: fmt.Sprintf("required field of type %s", nestedProp.Type),
						Actual:   "missing",
						Status:   "FAIL",
						Message:  "Required nested field missing",
					})
				}
				continue
			}

			actualType := getJSONType(val)
			expectedType := string(nestedProp.Type)
			if expectedType != actualType && expectedType != "unknown" {
				results = append(results, CheckResult{
					Resource: resourceName,
					Field:    path,
					Expected: fmt.Sprintf("type %s", expectedType),
					Actual:   fmt.Sprintf("%v (%s)", val, actualType),
					Status:   "FAIL",
					Message:  "Nested field type mismatch",
				})
			} else {
				results = append(results, CheckResult{
					Resource: resourceName,
					Field:    path,
					Expected: fmt.Sprintf("type %s", expectedType),
					Actual:   fmt.Sprintf("%v", val),
					Status:   "PASS",
				})
			}

			if nestedProp.Type == model.TypeObject && nestedProp.Properties != nil {
				if objData, ok := val.(map[string]interface{}); ok {
					subResults := checkObjectProperties(nestedProp, objData, path, resourceName)
					results = append(results, subResults...)
				}
			}
		}
	}
	return results
}
