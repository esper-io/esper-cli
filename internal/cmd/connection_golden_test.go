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

type connectionFixture struct {
	key, name, method, path, body string
	args                          []string
	query                         url.Values
	status, errorStatus           int
}

func connectionFixtures() []connectionFixture {
	return []connectionFixture{
		{"authn2 GET /authn2/v0/tenant/{enterprise_id}/connection", "connection-list", http.MethodGet, "/authn2/v0/tenant/tenant-1/connection", "", []string{"connection", "list", "--tenant", "tenant-1", "--is-default", "true", "--json"}, url.Values{"is_default": {"true"}}, http.StatusOK, http.StatusBadRequest},
		{"authn2 PUT /authn2/v0/tenant/{enterprise_id}/connection/{connection_id}", "custom-connection-update", http.MethodPut, "/authn2/v0/tenant/tenant-1/connection/connection-1", `{"display_name":"Acme SSO","strategy":"oidc","config":{"issuer_url":"https://login.example.test","client_id":"client-1"}}`, []string{"custom-connection", "update", "tenant-1", "connection-1", "--body", `{"display_name":"Acme SSO","strategy":"oidc","config":{"issuer_url":"https://login.example.test","client_id":"client-1"}}`, "--json"}, nil, http.StatusOK, http.StatusBadRequest},
	}
}

func TestConnectionOperationCoverage(t *testing.T) {
	rows, actual := map[string]bool{}, map[string]bool{}
	for _, row := range connectionFixtures() {
		if rows[row.key] {
			t.Fatalf("duplicate fixture row %s", row.key)
		}
		rows[row.key] = true
	}
	if len(rows) != 2 {
		t.Fatalf("fixture rows = %d, want 2", len(rows))
	}
	for _, operation := range generated.Operations() {
		if operation.Noun == "connection" || operation.Noun == "custom-connection" {
			actual[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if !reflect.DeepEqual(rows, actual) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(rows), len(actual))
	}
}

func TestConnectionFixturesMatchResponseContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "authn2.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, row := range connectionFixtures() {
		parts := strings.SplitN(row.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		fixtureSchemaValidateConnectionFile(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		value := fixtureSchemaValidateConnectionFile(t, document, fixtureSchemaResponse(t, document, operation, row.errorStatus), row.name+"-api-error.json")
		if object, ok := value.(map[string]any); !ok || object["code"] != float64(row.errorStatus) {
			t.Fatalf("%s error code = %v, want %d", row.name, object["code"], row.errorStatus)
		}
	}
}

func TestConnectionCommandsGoldenFixtures(t *testing.T) {
	for _, row := range connectionFixtures() {
		t.Run(row.name, func(t *testing.T) { executeConnectionFixture(t, row, false) })
	}
}

func TestConnectionListWithoutOptionalFilter(t *testing.T) {
	row := connectionFixtures()[0]
	row.args = []string{"connection", "list", "--tenant", "tenant-1", "--json"}
	row.query = nil
	executeConnectionFixture(t, row, false)
}

func TestConnectionCommandsAPIErrors(t *testing.T) {
	for _, row := range connectionFixtures() {
		t.Run(row.name, func(t *testing.T) { executeConnectionFixture(t, row, true) })
	}
}

func TestCustomConnectionUpdateBodyInput(t *testing.T) {
	found := false
	for _, operation := range generated.Operations() {
		if operation.Noun != "custom-connection" || operation.Verb != "update" {
			continue
		}
		found = true
		if operation.Body == nil || !operation.Body.Required || !operation.Body.BodyOnly || len(operation.Body.Properties) != 0 {
			t.Fatalf("body metadata = %#v", operation.Body)
		}
	}
	if !found {
		t.Fatal("custom-connection update operation not generated")
	}

	command := NewRootCommand()
	update, _, err := command.Find([]string{"custom-connection", "update"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"display-name", "strategy", "config"} {
		if update.Flags().Lookup(name) != nil {
			t.Fatalf("unexpected property flag --%s", name)
		}
	}

	assertConnectionUsage(t, []string{"custom-connection", "update", "tenant-1", "connection-1"})
	assertConnectionUsage(t, []string{"custom-connection", "update", "tenant-1", "connection-1", "--body", "{"})

	payload := `{"display_name":"Acme SSO","strategy":"oidc","config":{"issuer_url":"https://login.example.test","client_id":"client-1"}}`
	file := filepath.Join(t.TempDir(), "connection.json")
	if err := os.WriteFile(file, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{payload, "@" + file, "-"} {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			got, err := io.ReadAll(request.Body)
			if err != nil || string(got) != payload {
				t.Errorf("body = %q, error = %v", got, err)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(readConnectionFixture(t, "custom-connection-update-success.json"))
		}))
		command := configuredConnectionCommand(t, server.URL)
		command.SetIn(strings.NewReader(payload))
		command.SetArgs([]string{"custom-connection", "update", "tenant-1", "connection-1", "--body", body, "--json"})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		server.Close()
	}
}

func executeConnectionFixture(t *testing.T, row connectionFixture, apiError bool) {
	t.Helper()
	fixture, status := row.name+"-success.json", row.status
	if apiError {
		fixture, status = row.name+"-api-error.json", row.errorStatus
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != row.method || request.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, row.method, row.path)
		}
		if got, want := request.URL.Query().Encode(), row.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		if request.Header.Get("X-User-ID") != "" || request.Header.Get("X-Tenant-ID") != "" {
			t.Errorf("unexpected internal headers: X-User-ID=%q X-Tenant-ID=%q", request.Header.Get("X-User-ID"), request.Header.Get("X-Tenant-ID"))
		}
		if request.Header.Get("Accept") != "application/json" {
			t.Errorf("accept = %q, want application/json", request.Header.Get("Accept"))
		}
		body, err := io.ReadAll(request.Body)
		if err != nil || string(body) != row.body {
			t.Errorf("body = %q, error = %v, want %q", body, err, row.body)
		}
		if row.body != "" && request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q, want application/json", request.Header.Get("Content-Type"))
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(readConnectionFixture(t, fixture))
	}))
	defer server.Close()

	command := configuredConnectionCommand(t, server.URL)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(row.args)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != status || esperruntime.ExitCode(err) != 1 {
			t.Fatalf("API error = %v", err)
		}
		if want := readConnectionFixture(t, fixture); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %q, want %q", value.Body, want)
		}
		if output.Len() != 0 {
			t.Fatalf("API error output = %q, want empty", output.Bytes())
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if want := readConnectionFixture(t, row.name+"-success.golden"); !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.Bytes(), want)
	}
}

func fixtureSchemaValidateConnectionFile(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readConnectionFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

func assertConnectionUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}

func configuredConnectionCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func readConnectionFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "connection", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
