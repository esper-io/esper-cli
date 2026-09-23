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

type blueprintFixtureTest struct {
	key, name, method, path, body, fixture string
	arguments                              []string
	query                                  url.Values
	status, errorStatus                    int
	all, destructive                       bool
}

func blueprintFixtureTests() []blueprintFixtureTest {
	return []blueprintFixtureTest{
		{"v2 GET /v2/blueprints/", "blueprint list", http.MethodGet, "/v2/blueprints/", "", "v2-blueprint-list", []string{"blueprint", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusBadRequest, true, false},
		{"v2 POST /v2/blueprints/", "blueprint create", http.MethodPost, "/v2/blueprints/", `{"description":"Created through scalar flags","name":"New warehouse blueprint"}`, "v2-blueprint-create", []string{"blueprint", "create", "--name", "New warehouse blueprint", "--description", "Created through scalar flags", "--json"}, nil, http.StatusCreated, http.StatusBadRequest, false, false},
		{"v2 GET /v2/blueprints/{blueprint_id}/", "blueprint get", http.MethodGet, "/v2/blueprints/blueprint-1/", "", "v2-blueprint-get", []string{"blueprint", "get", "blueprint-1", "--json"}, nil, http.StatusOK, http.StatusNotFound, false, false},
		{"v2 PUT /v2/blueprints/{blueprint_id}/", "blueprint update", http.MethodPut, "/v2/blueprints/blueprint-1/", `{"name":"Updated warehouse blueprint"}`, "v2-blueprint-update", []string{"blueprint", "update", "blueprint-1", "--body", `{"name":"Updated warehouse blueprint"}`, "--json"}, nil, http.StatusOK, http.StatusBadRequest, false, false},
		{"v2 DELETE /v2/blueprints/{blueprint_id}/", "blueprint delete", http.MethodDelete, "/v2/blueprints/blueprint-1/", "", "v2-blueprint-delete", []string{"blueprint", "delete", "blueprint-1", "--yes", "--json"}, nil, http.StatusNoContent, http.StatusNotFound, false, true},
		{"v2 GET /v2/blueprints/{blueprint_id}/versions/{version_id}/", "blueprint-version get", http.MethodGet, "/v2/blueprints/blueprint-1/versions/version-1/", "", "v2-blueprint-version-get", []string{"blueprint-version", "get", "version-1", "--blueprint", "blueprint-1", "--json"}, nil, http.StatusOK, http.StatusNotFound, false, false},
	}
}

func TestBlueprintOperationCoverage(t *testing.T) {
	nouns := map[string]bool{"blueprint": true, "blueprint-version": true}
	expected := map[string]bool{}
	for _, test := range blueprintFixtureTests() {
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
		if nouns[operation.Noun] {
			actual[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if len(actual) != 6 || !reflect.DeepEqual(expected, actual) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(expected), len(actual))
	}
}

func TestBlueprintCommandsGoldenFixtures(t *testing.T) {
	for _, test := range blueprintFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executeBlueprintFixture(t, test, false) })
	}
}

func TestBlueprintCommandsAPIErrors(t *testing.T) {
	for _, test := range blueprintFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executeBlueprintFixture(t, test, true) })
	}
}

func TestBlueprintInputValidation(t *testing.T) {
	root := NewRootCommand()
	create, _, err := root.Find([]string{"blueprint", "create"})
	if err != nil || create.Flags().Lookup("publish") != nil {
		t.Fatal("blueprint create exposes read-only --publish")
	}
}

func TestBlueprintDestructiveCommandsDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	for _, test := range blueprintFixtureTests() {
		if !test.destructive {
			continue
		}
		command := NewRootCommand()
		command.SetIn(strings.NewReader("no\n"))
		command.SetArgs(withoutYes(test.arguments))
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) error = %v", test.arguments, err)
		}
	}
	if requests != 0 {
		t.Fatalf("declined destructive requests = %d, want 0", requests)
	}
}

func executeBlueprintFixture(t *testing.T, test blueprintFixtureTest, apiError bool) {
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
		} else if request.URL.Query().Encode() != test.query.Encode() {
			t.Errorf("query = %q, want %q", request.URL.Query(), test.query)
		}
		assertBlueprintBody(t, request, test)
		response := readBlueprintFixture(t, fixture)
		if test.all && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+test.path+"?offset=1"), 1)
		}
		if len(response) > 0 {
			writer.Header().Set("Content-Type", "application/json")
		}
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
		if !errors.As(err, &value) || value.StatusCode != status || esperruntime.ExitCode(err) != 1 {
			t.Fatalf("API error = %v", err)
		}
		if want := readBlueprintFixture(t, fixture); !bytes.Equal(value.Body, want) {
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
	golden := readBlueprintFixture(t, test.fixture+"-success.golden")
	if len(golden) == 0 {
		if output.Len() != 0 {
			t.Fatalf("output = %q, want empty", output.String())
		}
		return
	}
	var got, want any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(golden, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", golden, output.Bytes())
	}
}

func assertBlueprintBody(t *testing.T, request *http.Request, test blueprintFixtureTest) {
	t.Helper()
	if test.body == "" {
		return
	}
	got, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(test.body), &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("body = %s, want %s", got, test.body)
	}
}

func readBlueprintFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "blueprint", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
