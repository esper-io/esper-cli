package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/esper-io/esper-cli/internal/cmd/generated"
	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

type customActionFixtureTest struct {
	key, name, method, path, body, fixture string
	arguments                              []string
	query                                  url.Values
	status, errorStatus                    int
	all, destructive                       bool
}

func customActionFixtureTests() []customActionFixtureTest {
	return []customActionFixtureTest{
		{"v2 GET /v2/custom-actions/", "custom-action list", http.MethodGet, "/v2/custom-actions/", "", "v2-custom-action-list", []string{"custom-action", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusBadRequest, true, false},
		{"v2 POST /v2/custom-actions/", "custom-action create", http.MethodPost, "/v2/custom-actions/", customActionCreateBody, "v2-custom-action-create", []string{"custom-action", "create", "--body", customActionCreateBody, "--json"}, nil, http.StatusCreated, http.StatusBadRequest, false, false},
		{"v2 GET /v2/custom-actions/{custom_action_id}/", "custom-action get", http.MethodGet, "/v2/custom-actions/custom-action-1/", "", "v2-custom-action-get", []string{"custom-action", "get", "custom-action-1", "--json"}, nil, http.StatusOK, http.StatusNotFound, false, false},
		{"v2 PUT /v2/custom-actions/{custom_action_id}/", "custom-action update", http.MethodPut, "/v2/custom-actions/custom-action-1/", `{"name":"Restarted kiosk"}`, "v2-custom-action-update", []string{"custom-action", "update", "custom-action-1", "--name", "Restarted kiosk", "--json"}, nil, http.StatusOK, http.StatusBadRequest, false, false},
		{"v2 DELETE /v2/custom-actions/{custom_action_id}/", "custom-action delete", http.MethodDelete, "/v2/custom-actions/custom-action-1/", "", "v2-custom-action-delete", []string{"custom-action", "delete", "custom-action-1", "--yes", "--json"}, nil, http.StatusOK, http.StatusNotFound, false, true},
		{"v2 GET /v2/scripts/{script_id}/", "script get", http.MethodGet, "/v2/scripts/script-1/", "", "v2-script-get", []string{"script", "get", "script-1", "--json"}, nil, http.StatusOK, http.StatusNotFound, false, false},
	}
}

const customActionCreateBody = `{"name":"Restart kiosk","type":"button","state":"active","position_in_blueprints":"blueprints_scripts","position_in_device_settings":"device_quick_settings","properties":{"description":"Restart kiosk"},"options":[]}`

func TestCustomActionOperationCoverage(t *testing.T) {
	expected := map[string]bool{}
	for _, test := range customActionFixtureTests() {
		if expected[test.key] {
			t.Fatalf("duplicate explicit fixture row %s", test.key)
		}
		expected[test.key] = true
	}
	if len(expected) != 6 {
		t.Fatalf("fixture rows = %d, want 6", len(expected))
	}
	actual := map[string]bool{}
	for _, operation := range generated.Operations() {
		if operation.Noun == "custom-action" || operation.Noun == "script" {
			actual[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if len(actual) != 6 || !reflect.DeepEqual(expected, actual) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(expected), len(actual))
	}
}

func TestCustomActionCommandsGoldenFixtures(t *testing.T) {
	for _, test := range customActionFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executeCustomActionFixture(t, test, false) })
	}
}

func TestCustomActionCommandsAPIErrors(t *testing.T) {
	for _, test := range customActionFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executeCustomActionFixture(t, test, true) })
	}
}

func TestCustomActionFixturesMatchResponseContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "v2.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, test := range customActionFixtureTests() {
		parts := strings.SplitN(test.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		customActionFixtureSchemaValidateFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-success.json")
		if test.all {
			customActionFixtureSchemaValidateFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-second-page.json")
		}
		errorSchema := fixtureSchemaResponseForStatus(document, operation, test.errorStatus)
		value := customActionFixtureSchemaValidateFile(t, document, errorSchema, test.fixture+"-api-error.json")
		if object, ok := value.(map[string]any); !ok || object["code"] != float64(test.errorStatus) {
			t.Fatalf("%s API error code = %#v, want %d", test.fixture, value, test.errorStatus)
		}
	}
}

func TestCustomActionBodyInputRules(t *testing.T) {
	root := NewRootCommand()
	create, _, err := root.Find([]string{"custom-action", "create"})
	if err != nil || create.Flags().Lookup("name") != nil || create.Flags().Lookup("properties") != nil || create.Flags().Lookup("options") != nil {
		t.Fatalf("create flags do not enforce body-only metadata")
	}
	update, _, err := root.Find([]string{"custom-action", "update"})
	if err != nil || update.Flags().Lookup("name") == nil || update.Flags().Lookup("properties") != nil || update.Flags().Lookup("options") != nil {
		t.Fatalf("update flags do not expose only scalar properties")
	}

	assertCustomActionRequest(t, []string{"custom-action", "create", "--body", customActionCreateBody}, http.MethodPost, "/v2/custom-actions/", customActionCreateBody, strings.NewReader(""))
	file := filepath.Join(t.TempDir(), "create.json")
	if err := os.WriteFile(file, []byte(customActionCreateBody), 0o600); err != nil {
		t.Fatal(err)
	}
	assertCustomActionRequest(t, []string{"custom-action", "create", "--body", "@" + file}, http.MethodPost, "/v2/custom-actions/", customActionCreateBody, strings.NewReader(""))
	assertCustomActionRequest(t, []string{"custom-action", "create", "--body", "-"}, http.MethodPost, "/v2/custom-actions/", customActionCreateBody, strings.NewReader(customActionCreateBody))
	assertCustomActionRequest(t, []string{"custom-action", "update", "custom-action-1", "--body", `{"options":[]}`}, http.MethodPut, "/v2/custom-actions/custom-action-1/", `{"options":[]}`, strings.NewReader(""))

	for _, arguments := range [][]string{
		{"custom-action", "create"},
		{"custom-action", "create", "--body", "{"},
		{"custom-action", "update", "custom-action-1"},
		{"custom-action", "update", "custom-action-1", "--name", "Restarted kiosk", "--body", `{}`},
	} {
		command := NewRootCommand()
		command.SetArgs(arguments)
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) error = %v", arguments, err)
		}
	}
}

func TestCustomActionDestructiveCommandDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	command := NewRootCommand()
	command.SetIn(strings.NewReader("no\n"))
	command.SetArgs([]string{"custom-action", "delete", "custom-action-1"})
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("delete refusal error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("declined destructive requests = %d, want 0", requests)
	}
}

func executeCustomActionFixture(t *testing.T, test customActionFixtureTest, apiError bool) {
	t.Helper()
	requests := 0
	fixture, status := test.fixture+"-success.json", test.status
	if apiError {
		fixture, status = test.fixture+"-api-error.json", test.errorStatus
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != test.method || request.URL.Path != test.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, test.method, test.path)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		if test.all && requests == 2 {
			if got, want := request.URL.Query().Encode(), (url.Values{"offset": {"1"}}).Encode(); got != want {
				t.Errorf("second-page query = %q, want %q", got, want)
			}
			fixture = test.fixture + "-second-page.json"
		} else if got, want := request.URL.Query().Encode(), test.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		assertCustomActionBody(t, request, test.body)
		response := readCustomActionFixture(t, fixture)
		if test.all && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+test.path+"?offset=1"), 1)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(response)
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(test.arguments)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != status || esperruntime.ExitCode(err) != 1 || output.Len() != 0 {
			t.Fatalf("API error = %v, output = %q", err, output.String())
		}
		if want := readCustomActionFixture(t, fixture); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %s, want %s", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if test.all && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	var got, want any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(readCustomActionFixture(t, test.fixture+"-success.golden"), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", readCustomActionFixture(t, test.fixture+"-success.golden"), output.Bytes())
	}
}

func assertCustomActionRequest(t *testing.T, arguments []string, method, path, body string, input io.Reader) {
	t.Helper()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != method || request.URL.Path != path {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		assertCustomActionBody(t, request, body)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(readCustomActionFixture(t, "v2-custom-action-create-success.json"))
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	command := NewRootCommand()
	command.SetIn(input)
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute(%v): %v", arguments, err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func assertCustomActionBody(t *testing.T, request *http.Request, want string) {
	t.Helper()
	got, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if want == "" {
		if len(got) != 0 {
			t.Errorf("body = %q, want empty", got)
		}
		return
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func customActionFixtureSchemaValidateFile(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readCustomActionFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

func readCustomActionFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "custom-action", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
