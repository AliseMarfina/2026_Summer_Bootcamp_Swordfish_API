package comparator

import (
	"encoding/json"
	"fmt"

	"github.com/AliseMarfina/swordfish-verifier/internal/model"
)

type CheckResult struct {
	Resource  string `json:"resource"`
	Field     string `json:"field"`
	Expected  string `json:"expected"`
	Actual    string `json:"actual"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
}

func Compare(resourceName string, spec *model.Spec, actualJSON []byte) ([]CheckResult, error) {
	resource, ok := spec.Resources[resourceName]
	if !ok {
		return []CheckResult{{
			Resource:  resourceName,
			Status:    "NOT_SUPPORTED",
			Message:   "Resource not found in specification",
			ErrorCode: "ERR_UNSUPPORTED_RESOURCE",
		}}, nil
	}

	var actual map[string]interface{}
	if err := json.Unmarshal(actualJSON, &actual); err != nil {
		return nil, fmt.Errorf("failed to parse actual JSON: %w", err)
	}

	var results []CheckResult

	for actualField := range actual {
		if _, ok := resource.Properties[actualField]; !ok {
			results = append(results, CheckResult{
				Resource:  resourceName,
				Field:     actualField,
				Status:    "FAIL",
				Message:   "Field not defined in specification",
				ErrorCode: "ERR_UNEXPECTED_FIELD",
			})
		}
	}

	// Проверка полей из спецификации
	for fieldName, prop := range resource.Properties {
		results = append(results, checkProperty(fieldName, prop, actual, fieldName, resourceName)...)
	}

	return results, nil
}

func checkProperty(fieldName string, prop *model.Property, data map[string]interface{}, fieldPath, resourceName string) []CheckResult {
	var results []CheckResult
	val, exists := data[fieldName]

	if !exists {
		if prop.Required {
			results = append(results, CheckResult{
				Resource:  resourceName,
				Field:     fieldPath,
				Expected:  fmt.Sprintf("required field of type %s", prop.Type),
				Actual:    "missing",
				Status:    "FAIL",
				Message:   "Required field is missing",
				ErrorCode: "ERR_MISSING_REQUIRED",
			})
		}
		return results
	}

	if val == nil {
		if !prop.Nullable {
			results = append(results, CheckResult{
				Resource:  resourceName,
				Field:     fieldPath,
				Expected:  fmt.Sprintf("non-null %s", prop.Type),
				Actual:    "null",
				Status:    "FAIL",
				Message:   "Field cannot be null",
				ErrorCode: "ERR_NULLABLE_VIOLATION",
			})
		} else {
			results = append(results, CheckResult{
				Resource: resourceName,
				Field:    fieldPath,
				Status:   "PASS",
				Message:  "Nullable field is null",
			})
		}
		return results
	}

	actualType := getJSONType(val)
	expectedType := string(prop.Type)

	if expectedType != string(model.TypeUnknown) && expectedType != actualType {
		results = append(results, CheckResult{
			Resource:  resourceName,
			Field:     fieldPath,
			Expected:  fmt.Sprintf("type %s", expectedType),
			Actual:    fmt.Sprintf("%v (%s)", val, actualType),
			Status:    "FAIL",
			Message:   fmt.Sprintf("Type mismatch: expected %s, got %s", expectedType, actualType),
			ErrorCode: "ERR_TYPE_MISMATCH",
		})
		return results
	}

	if len(prop.Enum) > 0 {
		strVal, ok := val.(string)
		if !ok {
			results = append(results, CheckResult{
				Resource:  resourceName,
				Field:     fieldPath,
				Expected:  fmt.Sprintf("enum string one of %v", prop.Enum),
				Actual:    fmt.Sprintf("%v", val),
				Status:    "FAIL",
				Message:   "Enum value must be string",
				ErrorCode: "ERR_INVALID_ENUM",
			})
			return results
		}

		found := false
		for _, enumVal := range prop.Enum {
			if strVal == enumVal {
				found = true
				break
			}
		}

		if !found {
			results = append(results, CheckResult{
				Resource:  resourceName,
				Field:     fieldPath,
				Expected:  fmt.Sprintf("one of %v", prop.Enum),
				Actual:    strVal,
				Status:    "FAIL",
				Message:   "Value not in enum",
				ErrorCode: "ERR_INVALID_ENUM",
			})
			return results
		}
	}

	if prop.Type == model.TypeObject {
		results = append(results, CheckResult{
			Resource: resourceName,
			Field:    fieldPath,
			Expected: "object",
			Actual:   "object",
			Status:   "PASS",
		})

		objData, ok := val.(map[string]interface{})
		if !ok {
			return results
		}

		// проверка лишних вложенных полей
		for actualField := range objData {
			if _, ok := prop.Properties[actualField]; !ok {
				results = append(results, CheckResult{
					Resource:  resourceName,
					Field:     fieldPath + "." + actualField,
					Status:    "FAIL",
					Message:   "Nested field not defined in specification",
					ErrorCode: "ERR_UNEXPECTED_FIELD",
				})
			}
		}

		for nestedName, nestedProp := range prop.Properties {
			nestedPath := fieldPath + "." + nestedName
			results = append(results, checkProperty(nestedName, nestedProp, objData, nestedPath, resourceName)...)
		}

		return results
	}

	results = append(results, CheckResult{
		Resource: resourceName,
		Field:    fieldPath,
		Expected: fmt.Sprintf("type %s", expectedType),
		Actual:   fmt.Sprintf("%v", val),
		Status:   "PASS",
	})

	return results
}

func getJSONType(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
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
