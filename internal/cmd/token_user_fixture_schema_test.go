package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTokenUserFixturesMatchResponseContracts keeps the offline replay bodies
// aligned with the selected OpenAPI response schemas.
func TestTokenUserFixturesMatchResponseContracts(t *testing.T) {
	documents := map[string]map[string]any{}
	for generation, file := range map[string]string{
		"authn2": "authn2.yaml", "authz2": "authz2.yaml", "legacy": "legacy.yaml", "v0": "v0.yaml", "v1": "v1.yaml",
	} {
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

	for _, test := range tokenUserFixtures() {
		test := test
		t.Run(test.fixture, func(t *testing.T) {
			parts := strings.SplitN(test.key, " ", 3)
			if len(parts) != 3 {
				t.Fatalf("invalid operation key %q", test.key)
			}
			document := documents[parts[0]]
			operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
			if !test.raw {
				fixtureSchemaValidateTokenUserFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-success.json")
			}
			if test.all {
				fixtureSchemaValidateTokenUserFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-second-page.json")
			}
			errorSchema := fixtureSchemaResponseForStatus(document, operation, test.errorStatus)
			if errorSchema == nil {
				errorSchema = map[string]any{"$ref": "#/components/schemas/Error_400_Bad_Data"}
			}
			value := fixtureSchemaValidateTokenUserFile(t, document, errorSchema, test.fixture+"-api-error.json")
			if object, ok := value.(map[string]any); ok {
				for _, field := range []string{"code", "status"} {
					if code, exists := object[field].(float64); exists && code != float64(test.errorStatus) {
						t.Fatalf("%s top-level %s = %v, want HTTP status %d", test.fixture, field, code, test.errorStatus)
					}
				}
			}
		})
	}
}

func fixtureSchemaValidateTokenUserFile(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readTokenUserFixture(t, name)
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
