package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// TestAppApplicationFixturesMatchResponseContracts validates resolved refs,
// required fields, primitive/container types, and top-level error code/status consistency.
func TestAppApplicationFixturesMatchResponseContracts(t *testing.T) {
	documents := map[string]map[string]any{}
	for generation, file := range map[string]string{"legacy": "legacy.yaml", "v0": "v0.yaml", "v1": "v1.yaml", "v2": "v2.yaml"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", file))
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		documents[generation] = document
	}
	for _, test := range appApplicationFixtureTests() {
		parts := strings.SplitN(test.key, " ", 3)
		if len(parts) != 3 {
			t.Fatalf("invalid operation key %q", test.key)
		}
		document := documents[parts[0]]
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		fixtureSchemaValidateFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-success.json")
		if test.all {
			fixtureSchemaValidateFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-second-page.json")
		}
		errorSchema := fixtureSchemaResponseForStatus(document, operation, test.errorStatus)
		if errorSchema == nil {
			// These legacy contracts omit a per-operation 4xx response; replay uses
			// the canonical error component until the operation contract is expanded.
			errorSchema = map[string]any{"$ref": "#/components/schemas/Error_400_Bad_Data"}
		}
		errorValue := fixtureSchemaValidateFile(t, document, errorSchema, test.fixture+"-api-error.json")
		if object, ok := errorValue.(map[string]any); ok {
			if code, exists := object["code"]; exists {
				if code != float64(test.errorStatus) {
					t.Fatalf("%s top-level code = %v, want HTTP status %d", test.fixture, code, test.errorStatus)
				}
			}
		}
	}
}

func fixtureSchemaOperation(t *testing.T, document map[string]any, path, method string) map[string]any {
	t.Helper()
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatal("OpenAPI document has no paths object")
	}
	pathItem, ok := paths[path].(map[string]any)
	if !ok {
		t.Fatalf("missing schema path %s", path)
	}
	operation, ok := pathItem[method].(map[string]any)
	if !ok {
		t.Fatalf("missing schema operation %s %s", method, path)
	}
	return operation
}

func fixtureSchemaResponse(t *testing.T, document, operation map[string]any, status int) map[string]any {
	t.Helper()
	schema := fixtureSchemaResponseForStatus(document, operation, status)
	if schema == nil && status == 204 {
		return nil
	}
	if schema == nil {
		t.Fatalf("missing success schema for status %d", status)
	}
	return schema
}

func fixtureSchemaResponseForStatus(document, operation map[string]any, status int) map[string]any {
	responses, ok := operation["responses"].(map[string]any)
	if !ok {
		return nil
	}
	response, ok := responses[strconv.Itoa(status)].(map[string]any)
	if !ok {
		response, ok = responses["default"].(map[string]any)
	}
	if !ok {
		return nil
	}
	content, ok := response["content"].(map[string]any)
	if !ok {
		return nil
	}
	for _, media := range content {
		mediaObject, ok := media.(map[string]any)
		if !ok {
			continue
		}
		if schema, ok := mediaObject["schema"].(map[string]any); ok {
			return schema
		}
	}
	return nil
}

func fixtureSchemaValidateFile(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readAppApplicationFixture(t, name)
	if schema == nil {
		if len(data) != 0 {
			t.Fatalf("%s has unexpected response body", name)
		}
		return nil
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

func fixtureSchemaValidate(document, schema map[string]any, value any, path string, resolving map[string]bool) error {
	if ref, _ := schema["$ref"].(string); ref != "" {
		if resolving[ref] {
			return nil
		}
		resolving[ref] = true
		defer delete(resolving, ref)
		name := strings.TrimPrefix(ref, "#/components/schemas/")
		components, ok := document["components"].(map[string]any)
		if !ok {
			return fmt.Errorf("%s has no components object", path)
		}
		schemas, ok := components["schemas"].(map[string]any)
		if !ok {
			return fmt.Errorf("%s has no schemas object", path)
		}
		target, ok := schemas[name].(map[string]any)
		if !ok {
			return fmt.Errorf("%s references missing schema %s", path, ref)
		}
		return fixtureSchemaValidate(document, target, value, path, resolving)
	}
	for _, key := range []string{"allOf", "oneOf", "anyOf"} {
		if options, ok := schema[key].([]any); ok && len(options) > 0 {
			if key == "allOf" {
				for _, option := range options {
					child, ok := option.(map[string]any)
					if !ok {
						return fmt.Errorf("%s has invalid %s option", path, key)
					}
					if err := fixtureSchemaValidate(document, child, value, path, resolving); err != nil {
						return err
					}
				}
				return nil
			}
			for _, option := range options {
				child, ok := option.(map[string]any)
				if !ok {
					continue
				}
				if fixtureSchemaValidate(document, child, value, path, resolving) == nil {
					return nil
				}
			}
			return fmt.Errorf("%s does not match %s", path, key)
		}
	}
	if values, ok := schema["enum"].([]any); ok {
		for _, candidate := range values {
			if reflect.DeepEqual(candidate, value) {
				return nil
			}
		}
		return fmt.Errorf("%s must be one of %v", path, values)
	}
	types := fixtureSchemaTypes(schema["type"])
	if len(types) == 0 {
		return nil
	}
	if value == nil {
		for _, typeName := range types {
			if typeName == "null" {
				return nil
			}
		}
		return fmt.Errorf("%s must not be null", path)
	}
	for _, typeName := range types {
		if fixtureSchemaTypeMatches(typeName, value) {
			return fixtureSchemaValidateType(document, schema, value, path, resolving, typeName)
		}
	}
	return fmt.Errorf("%s must be one of %s", path, strings.Join(types, ", "))
}

func fixtureSchemaValidateType(document, schema map[string]any, value any, path string, resolving map[string]bool, typeName string) error {
	switch typeName {
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object", path)
		}
		for _, required := range stringSlice(schema["required"]) {
			if _, ok := object[required]; !ok {
				return fmt.Errorf("%s.%s is required", path, required)
			}
		}
		properties, _ := schema["properties"].(map[string]any)
		for name, child := range properties {
			if childValue, ok := object[name]; ok {
				childSchema, ok := child.(map[string]any)
				if !ok {
					return fmt.Errorf("%s.%s has invalid property schema", path, name)
				}
				if err := fixtureSchemaValidate(document, childSchema, childValue, path+"."+name, resolving); err != nil {
					return err
				}
			}
		}
	case "array":
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s must be an array", path)
		}
		if maxItems, ok := schema["maxItems"].(float64); ok && len(items) > int(maxItems) {
			return fmt.Errorf("%s must contain at most %d items", path, int(maxItems))
		}
		if item, ok := schema["items"].(map[string]any); ok {
			for index, child := range items {
				if err := fixtureSchemaValidate(document, item, child, fmt.Sprintf("%s[%d]", path, index), resolving); err != nil {
					return err
				}
			}
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", path)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", path)
		}
	case "integer", "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("%s must be a number", path)
		}
	}
	return nil
}

func fixtureSchemaTypes(value any) []string {
	if typeName, ok := value.(string); ok {
		return []string{typeName}
	}
	values, _ := value.([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if typeName, ok := value.(string); ok {
			result = append(result, typeName)
		}
	}
	return result
}

func fixtureSchemaTypeMatches(typeName string, value any) bool {
	switch typeName {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer", "number":
		_, ok := value.(float64)
		return ok
	}
	return false
}

func stringSlice(value any) []string {
	values, _ := value.([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, text)
		}
	}
	return result
}
