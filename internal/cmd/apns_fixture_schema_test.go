package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPNsFixturesMatchResponseContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "v0.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, row := range apnsFixtures() {
		parts := strings.SplitN(row.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		if !row.raw {
			fixtureSchemaValidateAPNsFile(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.fixture+"-success.json")
		}
		value := fixtureSchemaValidateAPNsFile(t, document, fixtureSchemaResponse(t, document, operation, row.errorStatus), row.fixture+"-api-error.json")
		if object, ok := value.(map[string]any); ok && object["code"] != float64(row.errorStatus) {
			t.Fatalf("%s code = %v, want %d", row.fixture, object["code"], row.errorStatus)
		}
	}
}

func fixtureSchemaValidateAPNsFile(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readAPNsFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}
