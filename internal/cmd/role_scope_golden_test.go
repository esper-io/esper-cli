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
	"github.com/spf13/cobra"
)

const (
	roleScopeRoleID    = "11111111-1111-4111-8111-111111111111"
	roleScopeScopeID   = "22222222-2222-4222-8222-222222222222"
	roleScopeCreate    = `{"description":"Operates kiosk devices","name":"Kiosk Operator"}`
	roleScopeUpdate    = `{"description":"Updated kiosk operator"}`
	roleScopeScopeBody = `{"scope_ids":["22222222-2222-4222-8222-222222222222"]}`
)

type roleScopeFixture struct {
	id, name, method, path, body string
	args                         []string
	query                        url.Values
	status, errorStatus          int
}

func roleScopeFixtures() []roleScopeFixture {
	return []roleScopeFixture{
		{"GetRoleURL", "role-list", http.MethodGet, "/authz2/v1/roles/", "", []string{"role", "list", "--name", "Kiosk", "--role-type", "2", "--limit", "1", "--offset", "0", "--sort", "name", "--json"}, url.Values{"name": {"Kiosk"}, "role_type": {"2"}, "limit": {"1"}, "offset": {"0"}, "sort": {"name"}}, 200, 400},
		{"PostRoleURL", "role-create", http.MethodPost, "/authz2/v1/roles/", roleScopeCreate, []string{"role", "create", "--name", "Kiosk Operator", "--description", "Operates kiosk devices", "--json"}, nil, 201, 400},
		{"GetRoleIdURL", "role-get", http.MethodGet, "/authz2/v1/roles/" + roleScopeRoleID, "", []string{"role", "get", roleScopeRoleID, "--json"}, nil, 200, 404},
		{"UpdateRoleIdURL", "role-update", http.MethodPatch, "/authz2/v1/roles/" + roleScopeRoleID, roleScopeUpdate, []string{"role", "update", roleScopeRoleID, "--description", "Updated kiosk operator", "--json"}, nil, 200, 404},
		{"DeleteRoleIdURL", "role-delete", http.MethodDelete, "/authz2/v1/roles/" + roleScopeRoleID, "", []string{"role", "delete", roleScopeRoleID, "--yes"}, nil, 204, 400},
		{"GetRoleScopeURL", "scope-list", http.MethodGet, "/authz2/v1/roles/" + roleScopeRoleID + "/scopes", "", []string{"scope", "list", "--role", roleScopeRoleID, "--json"}, nil, 200, 404},
		{"PutRoleScopeURL", "scope-update", http.MethodPut, "/authz2/v1/roles/" + roleScopeRoleID + "/scopes", roleScopeScopeBody, []string{"scope", "update", "--role", roleScopeRoleID, "--body", roleScopeScopeBody, "--json"}, nil, 200, 404},
	}
}

func TestRoleScopeOperationCoverage(t *testing.T) {
	want := map[string]struct{ method, path, pagination, parent string }{
		"GetRoleURL":      {"GET", "/authz2/v1/roles/", "none", ""},
		"PostRoleURL":     {"POST", "/authz2/v1/roles/", "none", ""},
		"GetRoleIdURL":    {"GET", "/authz2/v1/roles/{role_id}", "none", ""},
		"UpdateRoleIdURL": {"PATCH", "/authz2/v1/roles/{role_id}", "none", ""},
		"DeleteRoleIdURL": {"DELETE", "/authz2/v1/roles/{role_id}", "none", ""},
		"GetRoleScopeURL": {"GET", "/authz2/v1/roles/{role_id}/scopes", "none", "role"},
		"PutRoleScopeURL": {"PUT", "/authz2/v1/roles/{role_id}/scopes", "none", "role"},
	}
	got := map[string]struct{ method, path, pagination, parent string }{}
	for _, operation := range generated.Operations() {
		if _, ok := want[operation.OperationID]; ok {
			got[operation.OperationID] = struct{ method, path, pagination, parent string }{operation.Method, operation.Path, operation.Pagination, operation.ScopeParent}
		}
	}
	if len(want) != 7 || !reflect.DeepEqual(want, got) {
		t.Fatalf("operations = %#v", got)
	}
}

func TestRoleScopeFixturesMatchContracts(t *testing.T) {
	document := readRoleScopeDocument(t)
	for _, row := range roleScopeFixtures() {
		operation := fixtureSchemaOperation(t, document, roleScopeOperation(t, row.id).Path, strings.ToLower(row.method))
		if row.status == http.StatusNoContent {
			if got := readRoleScopeFixture(t, row.name+"-success.json"); len(got) != 0 {
				t.Fatalf("%s 204 fixture = %q", row.name, got)
			}
		} else {
			roleScopeValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		}
		value := roleScopeValidateFixture(t, document, fixtureSchemaResponseForStatus(document, operation, row.errorStatus), row.name+"-api-error.json")
		if object, ok := value.(map[string]any); !ok || object["status"] != float64(row.errorStatus) {
			t.Fatalf("%s error = %#v", row.name, value)
		}
	}
}

func TestRoleScopeFixtureInventory(t *testing.T) {
	want := map[string]bool{"README.md": true, "RERECORD_WITH_TENANT": true}
	for _, row := range roleScopeFixtures() {
		for _, suffix := range []string{"-success.json", "-success.golden", "-api-error.json"} {
			want[row.name+suffix] = true
		}
	}
	entries, err := os.ReadDir(roleScopeFixtureDir())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, entry := range entries {
		got[entry.Name()] = true
	}
	if len(want) != 23 || !reflect.DeepEqual(want, got) {
		t.Fatalf("fixture inventory = %#v", got)
	}
}

func TestRoleScopeGoldenFixtures(t *testing.T) {
	for _, row := range roleScopeFixtures() {
		t.Run(row.name, func(t *testing.T) { executeRoleScopeFixture(t, row, false) })
	}
}

func TestRoleScopeAPIErrors(t *testing.T) {
	for _, row := range roleScopeFixtures() {
		t.Run(row.name, func(t *testing.T) { executeRoleScopeFixture(t, row, true) })
	}
}

func TestRoleScopeInputRules(t *testing.T) {
	root := NewRootCommand()
	for _, path := range [][]string{{"scope", "list"}, {"scope", "update"}} {
		command, _, err := root.Find(path)
		if err != nil || command.Flags().Lookup("role") == nil {
			t.Fatalf("%s requires --role: %v", strings.Join(path, " "), err)
		}
	}
	scopeUpdate, _, _ := root.Find([]string{"scope", "update"})
	for _, name := range []string{"scope-ids", "scope-names"} {
		if scopeUpdate.Flags().Lookup(name) != nil {
			t.Fatalf("scope update exposes --%s", name)
		}
	}
	for _, path := range [][]string{{"role", "list"}, {"role", "create"}, {"role", "get"}, {"role", "update"}, {"role", "delete"}, {"scope", "list"}, {"scope", "update"}} {
		command, _, _ := root.Find(path)
		for _, name := range []string{"x-esper-tenant-id", "x-esper-user-id"} {
			if command.Flags().Lookup(name) != nil {
				t.Fatalf("%s exposes --%s", strings.Join(path, " "), name)
			}
		}
	}
	for _, args := range [][]string{
		{"role", "create"},
		{"role", "create", "--name", "Kiosk Operator"},
		{"role", "create", "--body", roleScopeCreate, "--name", "Kiosk Operator"},
		{"role", "update", roleScopeRoleID},
		{"role", "update", roleScopeRoleID, "--body", "{"},
		{"role", "update", roleScopeRoleID, "--body", roleScopeUpdate, "--description", "Updated kiosk operator"},
		{"scope", "update", "--role", roleScopeRoleID},
		{"scope", "update", "--role", roleScopeRoleID, "--body", "{"},
		{"scope", "update", "--body", roleScopeScopeBody},
		{"scope", "list"},
	} {
		assertRoleScopeUsage(t, args)
	}
	assertRoleScopeRequest(t, []string{"role", "update", roleScopeRoleID, "--name", "Renamed"}, `{"name":"Renamed"}`, roleScopeFixture{method: http.MethodPatch, path: "/authz2/v1/roles/" + roleScopeRoleID, status: 200})
	assertRoleScopeBodyModes(t, []string{"scope", "update", "--role", roleScopeRoleID}, roleScopeScopeBody, roleScopeFixture{method: http.MethodPut, path: "/authz2/v1/roles/" + roleScopeRoleID + "/scopes", status: 200})
	assertRoleScopeBodyModes(t, []string{"role", "update", roleScopeRoleID}, roleScopeUpdate, roleScopeFixture{method: http.MethodPatch, path: "/authz2/v1/roles/" + roleScopeRoleID, status: 200})
	assertRoleScopeBodyModes(t, []string{"role", "create"}, roleScopeCreate, roleScopeFixture{method: http.MethodPost, path: "/authz2/v1/roles/", status: 201})
}

func TestRoleScopeDeleteDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	command := configuredRoleScopeCommand(t, server.URL)
	command.SetIn(strings.NewReader("no\n"))
	command.SetArgs([]string{"role", "delete", roleScopeRoleID})
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 || requests != 0 {
		t.Fatalf("delete refusal = %v, requests = %d", err, requests)
	}
}

func executeRoleScopeFixture(t *testing.T, row roleScopeFixture, apiError bool) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != row.method || request.URL.Path != row.path || request.URL.Query().Encode() != row.query.Encode() {
			t.Errorf("request = %s %s?%s", request.Method, request.URL.Path, request.URL.Query())
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" || request.Header.Get("Accept") != "application/json" {
			t.Errorf("headers = %#v", request.Header)
		}
		for _, name := range []string{"X-Esper-Tenant-ID", "X-Esper-User-ID"} {
			if request.Header.Get(name) != "" {
				t.Errorf("internal header %s was sent", name)
			}
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != row.body {
			t.Errorf("body = %q, want %q", body, row.body)
		}
		if row.body != "" && request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q", request.Header.Get("Content-Type"))
		}
		name, status := row.name+"-success.json", row.status
		if apiError {
			name, status = row.name+"-api-error.json", row.errorStatus
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(readRoleScopeFixture(t, name))
	}))
	defer server.Close()
	command := configuredRoleScopeCommand(t, server.URL)
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs(row.args)
	err := command.Execute()
	if apiError {
		var api *esperruntime.APIError
		if !errors.As(err, &api) || api.StatusCode != row.errorStatus || esperruntime.ExitCode(err) != 1 || output.Len() != 0 || stderr.Len() != 0 || !bytes.Equal(api.Body, readRoleScopeFixture(t, row.name+"-api-error.json")) {
			t.Fatalf("API error = %v stdout=%q stderr=%q", err, output.Bytes(), stderr.Bytes())
		}
		return
	}
	if err != nil || stderr.Len() != 0 || !bytes.Equal(output.Bytes(), readRoleScopeFixture(t, row.name+"-success.golden")) {
		t.Fatalf("error=%v stdout=%q stderr=%q", err, output.Bytes(), stderr.Bytes())
	}
}

func assertRoleScopeUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}

func assertRoleScopeRequest(t *testing.T, args []string, want string, row roleScopeFixture) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		if request.Method != row.method || request.URL.Path != row.path || string(body) != want {
			t.Errorf("request = %s %s %q", request.Method, request.URL.Path, body)
		}
		w.WriteHeader(row.status)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	command := configuredRoleScopeCommand(t, server.URL)
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
}

func assertRoleScopeBodyModes(t *testing.T, prefix []string, body string, row roleScopeFixture) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{body, "@" + file, "-"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			got, _ := io.ReadAll(request.Body)
			if request.Method != row.method || request.URL.Path != row.path || string(got) != body {
				t.Errorf("request = %s %s %q, want %s %s %q", request.Method, request.URL.Path, got, row.method, row.path, body)
			}
			if request.Header.Get("Content-Type") != "application/json" {
				t.Errorf("content type = %q", request.Header.Get("Content-Type"))
			}
			w.WriteHeader(row.status)
			_, _ = w.Write([]byte(`{}`))
		}))
		command := configuredRoleScopeCommand(t, server.URL)
		if value == "-" {
			command.SetIn(strings.NewReader(body))
		}
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)
		command.SetArgs(append(append([]string{}, prefix...), "--body", value))
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		server.Close()
	}
}

func configuredRoleScopeCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func roleScopeOperation(t *testing.T, id string) generated.Operation {
	t.Helper()
	for _, operation := range generated.Operations() {
		if operation.OperationID == id {
			return operation
		}
	}
	t.Fatalf("operation %s not found", id)
	return generated.Operation{}
}

func roleScopeFixtureDir() string { return filepath.Join("..", "..", "spec", "fixtures", "role-scope") }

func readRoleScopeFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(roleScopeFixtureDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readRoleScopeDocument(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "authz2.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func roleScopeValidateFixture(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal(readRoleScopeFixture(t, name), &value); err != nil {
		t.Fatal(err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}
