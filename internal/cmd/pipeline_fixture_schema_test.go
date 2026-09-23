package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipelineFixturesMatchResponseContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "pipelines-v0.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, test := range pipelineFixtureTests() {
		parts := strings.SplitN(test.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		fixtureSchemaValidatePipelineFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-success.json")
		if test.all {
			fixtureSchemaValidatePipelineFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-second-page.json")
		}
		errorSchema := fixtureSchemaResponseForStatus(document, operation, test.errorStatus)
		if errorSchema == nil {
			errorSchema = map[string]any{"$ref": "#/components/schemas/Error_400_Bad_Data"}
		}
		value := fixtureSchemaValidatePipelineFile(t, document, errorSchema, test.fixture+"-api-error.json")
		if object, ok := value.(map[string]any); ok {
			if code, ok := object["code"].(float64); ok && code != float64(test.errorStatus) {
				t.Fatalf("%s top-level code = %v, want HTTP status %d", test.fixture, code, test.errorStatus)
			}
		}
	}
}

func fixtureSchemaValidatePipelineFile(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readPipelineFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}
