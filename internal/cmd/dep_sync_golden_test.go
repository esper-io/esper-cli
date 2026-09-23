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

type depSyncFixtureTest struct {
	key, name, method, path, body, fixture string
	arguments                              []string
	query                                  url.Values
	status, errorStatus                    int
	all                                    bool
}

func depSyncFixtureTests() []depSyncFixtureTest {
	return []depSyncFixtureTest{
		{"v0 GET /onboarding/v0/depsyncs/", "dep-sync-request list", http.MethodGet, "/onboarding/v0/depsyncs/", "", "dep-sync-request-list", []string{"dep-sync-request", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusInternalServerError, true},
		{"v0 POST /onboarding/v0/depsyncs/", "dep-sync-request create", http.MethodPost, "/onboarding/v0/depsyncs/", "{}", "dep-sync-request-create", []string{"dep-sync-request", "create", "--json"}, nil, http.StatusOK, http.StatusBadRequest, false},
		{"v0 GET /onboarding/v0/depsyncs/{id}/", "dep-sync-request get", http.MethodGet, "/onboarding/v0/depsyncs/dep-sync-1/", "", "dep-sync-request-get", []string{"dep-sync-request", "get", "dep-sync-1", "--json"}, nil, http.StatusOK, http.StatusNotFound, false},
	}
}

func TestDepSyncOperationCoverage(t *testing.T) {
	expected, actual := map[string]bool{}, map[string]bool{}
	for _, test := range depSyncFixtureTests() {
		if expected[test.key] {
			t.Fatalf("duplicate explicit fixture row %s", test.key)
		}
		expected[test.key] = true
	}
	if len(expected) != 3 {
		t.Fatalf("fixture rows = %d, want 3", len(expected))
	}
	for _, operation := range generated.Operations() {
		if operation.Noun == "dep-sync-request" {
			actual[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(expected), len(actual))
	}
}

func TestDepSyncCommandsGoldenFixtures(t *testing.T) {
	for _, test := range depSyncFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executeDepSyncFixture(t, test, false) })
	}
}

func TestDepSyncCommandsAPIErrors(t *testing.T) {
	for _, test := range depSyncFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executeDepSyncFixture(t, test, true) })
	}
}

func TestDepSyncFixturesMatchResponseContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "v0.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, test := range depSyncFixtureTests() {
		parts := strings.SplitN(test.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		depSyncFixtureSchemaValidateFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-success.json")
		if test.all {
			depSyncFixtureSchemaValidateFile(t, document, fixtureSchemaResponse(t, document, operation, test.status), test.fixture+"-second-page.json")
		}
		value := depSyncFixtureSchemaValidateFile(t, document, fixtureSchemaResponse(t, document, operation, test.errorStatus), test.fixture+"-api-error.json")
		if object, ok := value.(map[string]any); !ok || object["code"] != float64(test.errorStatus) {
			t.Fatalf("%s API error code = %#v, want %d", test.fixture, value, test.errorStatus)
		}
	}
}

func TestDepSyncCreateEmptyRequiredBody(t *testing.T) {
	found := false
	for _, operation := range generated.Operations() {
		if operation.Noun != "dep-sync-request" || operation.Verb != "create" {
			continue
		}
		found = true
		if operation.Body == nil || !operation.Body.Required || !operation.Body.Empty || len(operation.Body.Properties) != 0 {
			t.Fatalf("body metadata = %#v", operation.Body)
		}
	}
	if !found {
		t.Fatal("dep-sync-request create operation not generated")
	}
	root := NewRootCommand()
	create, _, err := root.Find([]string{"dep-sync-request", "create"})
	if err != nil || create.Flags().Lookup("body") == nil {
		t.Fatalf("create body flag = %v, error = %v", create.Flags().Lookup("body"), err)
	}
}

func executeDepSyncFixture(t *testing.T, test depSyncFixtureTest, apiError bool) {
	t.Helper()
	fixture, status := test.fixture+"-success.json", test.status
	all := test.all && !apiError
	if apiError {
		fixture, status = test.fixture+"-api-error.json", test.errorStatus
	}
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != test.method || request.URL.Path != test.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, test.method, test.path)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		if all && requests == 2 {
			if got, want := request.URL.Query().Encode(), (url.Values{"offset": {"1"}}).Encode(); got != want {
				t.Errorf("second-page query = %q, want %q", got, want)
			}
			fixture = test.fixture + "-second-page.json"
		} else if got, want := request.URL.Query().Encode(), test.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		assertDepSyncBody(t, request, test.body)
		response := readDepSyncFixture(t, fixture)
		if all && requests == 1 {
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
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	arguments := test.arguments
	if apiError && test.all {
		arguments = append([]string(nil), test.arguments[:len(test.arguments)-2]...)
		arguments = append(arguments, "--json")
	}
	command.SetArgs(arguments)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != status || esperruntime.ExitCode(err) != 1 || output.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("API error = %v, stdout = %q, stderr = %q", err, output.String(), stderr.String())
		}
		if want := readDepSyncFixture(t, fixture); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %s, want %s", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if all && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if want := readDepSyncFixture(t, test.fixture+"-success.golden"); !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.Bytes(), want)
	}
}

func assertDepSyncBody(t *testing.T, request *http.Request, want string) {
	t.Helper()
	got, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func depSyncFixtureSchemaValidateFile(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readDepSyncFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

func readDepSyncFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "dep-sync", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
