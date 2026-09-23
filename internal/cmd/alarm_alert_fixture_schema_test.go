package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAlarmAlertFixturesMatchResponseContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "v1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, test := range alarmAlertFixtures() {
		parts := strings.SplitN(test.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		fixtureSchemaValidateAlarmAlertFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-success.json")
		if test.all {
			fixtureSchemaValidateAlarmAlertFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-second-page.json")
		}
		value := fixtureSchemaValidateAlarmAlertFile(t, document, fixtureSchemaResponse(t, document, operation, test.errorStatus), test.fixture+"-api-error.json")
		if object, ok := value.(map[string]any); ok {
			if status, exists := object["status"].(float64); exists && status != float64(test.errorStatus) {
				t.Fatalf("%s top-level status = %v, want HTTP status %d", test.fixture, status, test.errorStatus)
			}
		}
	}
}

func fixtureSchemaValidateAlarmAlertFile(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readAlarmAlertFixture(t, name)
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
