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
	directoryRecordEnterpriseID = "11111111-1111-4111-8111-111111111111"
	directoryRecordID           = "22222222-2222-4222-8222-222222222222"
	directoryRecordBody         = `{"unique_identifiers":[{"unique_id":"SERIAL-001","type":2}],"group_id":"33333333-3333-4333-8333-333333333333","alias":"Lobby kiosk","tags":["lobby","production"]}`
)

type directoryRecordFixture struct {
	key, name, method, path, body string
	args                          []string
	query                         url.Values
	status, errorStatus           int
	all, destructive              bool
}

func directoryRecordFixtures() []directoryRecordFixture {
	return []directoryRecordFixture{
		{"v1 GET /v1/enterprise/{enterprise_id}/directory_record/", "v1-directory-record-list", http.MethodGet, "/v1/enterprise/" + directoryRecordEnterpriseID + "/directory_record/", "", []string{"directory-record", "list", "--enterprise", directoryRecordEnterpriseID, "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusBadRequest, true, false},
		{"v1 POST /v1/enterprise/{enterprise_id}/directory_record/", "v1-directory-record-create", http.MethodPost, "/v1/enterprise/" + directoryRecordEnterpriseID + "/directory_record/", directoryRecordBody, []string{"directory-record", "create", "--enterprise", directoryRecordEnterpriseID, "--body", directoryRecordBody, "--json"}, nil, http.StatusCreated, http.StatusBadRequest, false, false},
		{"v1 GET /v1/enterprise/{enterprise_id}/directory_record/{directory_record_id}/", "v1-directory-record-get", http.MethodGet, "/v1/enterprise/" + directoryRecordEnterpriseID + "/directory_record/" + directoryRecordID + "/", "", []string{"directory-record", "get", directoryRecordEnterpriseID, directoryRecordID, "--json"}, nil, http.StatusOK, http.StatusNotFound, false, false},
		{"v1 PUT /v1/enterprise/{enterprise_id}/directory_record/{directory_record_id}/", "v1-directory-record-update", http.MethodPut, "/v1/enterprise/" + directoryRecordEnterpriseID + "/directory_record/" + directoryRecordID + "/", directoryRecordBody, []string{"directory-record", "update", directoryRecordEnterpriseID, directoryRecordID, "--body", directoryRecordBody, "--json"}, nil, http.StatusOK, http.StatusBadRequest, false, false},
		{"v1 DELETE /v1/enterprise/{enterprise_id}/directory_record/{directory_record_id}/", "v1-directory-record-delete", http.MethodDelete, "/v1/enterprise/" + directoryRecordEnterpriseID + "/directory_record/" + directoryRecordID + "/", "", []string{"directory-record", "delete", directoryRecordEnterpriseID, directoryRecordID, "--yes"}, nil, http.StatusNoContent, http.StatusUnauthorized, false, true},
	}
}

func TestDirectoryRecordOperationCoverage(t *testing.T) {
	want, got := map[string]bool{}, map[string]bool{}
	for _, row := range directoryRecordFixtures() {
		if want[row.key] {
			t.Fatalf("duplicate fixture row %s", row.key)
		}
		want[row.key] = true
	}
	if len(want) != 5 {
		t.Fatalf("fixture rows = %d, want 5", len(want))
	}
	for _, operation := range generated.Operations() {
		if operation.Noun == "directory-record" {
			got[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(want), len(got))
	}
}

func TestDirectoryRecordFixturesMatchResponseContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "v1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, row := range directoryRecordFixtures() {
		parts := strings.SplitN(row.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		if row.status != http.StatusNoContent {
			directoryRecordValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		} else if data := readDirectoryRecordFixture(t, row.name+"-success.json"); len(data) != 0 {
			t.Fatalf("%s 204 response fixture = %q, want empty", row.name, data)
		}
		if row.all {
			directoryRecordValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-second-page.json")
		}
		value := directoryRecordValidateFixture(t, document, fixtureSchemaResponseForStatus(document, operation, row.errorStatus), row.name+"-api-error.json")
		if object, ok := value.(map[string]any); !ok || object["status"] != float64(row.errorStatus) {
			t.Fatalf("%s API error status = %#v, want %d", row.name, value, row.errorStatus)
		}
	}
}

func TestDirectoryRecordGoldenFixtures(t *testing.T) {
	for _, row := range directoryRecordFixtures() {
		t.Run(row.name, func(t *testing.T) { executeDirectoryRecordFixture(t, row, false) })
	}
}

func TestDirectoryRecordAPIErrors(t *testing.T) {
	for _, row := range directoryRecordFixtures() {
		t.Run(row.name, func(t *testing.T) { executeDirectoryRecordFixture(t, row, true) })
	}
}

func TestDirectoryRecordInputRules(t *testing.T) {
	root := NewRootCommand()
	for _, commandPath := range [][]string{{"directory-record", "create"}, {"directory-record", "update"}} {
		command, _, err := root.Find(commandPath)
		if err != nil || command.Flags().Lookup("body") == nil {
			t.Fatalf("%s body flag = %v, error = %v", strings.Join(commandPath, " "), command.Flags().Lookup("body"), err)
		}
		for _, name := range []string{"alias", "group-id", "tags", "unique-identifiers"} {
			if command.Flags().Lookup(name) != nil {
				t.Fatalf("%s unexpectedly exposes --%s", strings.Join(commandPath, " "), name)
			}
		}
	}

	for _, args := range [][]string{
		{"directory-record", "list"},
		{"directory-record", "create", "--body", directoryRecordBody},
		{"directory-record", "create", "--enterprise", directoryRecordEnterpriseID},
		{"directory-record", "create", "--enterprise", directoryRecordEnterpriseID, "--body", "{"},
		{"directory-record", "update", directoryRecordEnterpriseID, directoryRecordID},
		{"directory-record", "update", directoryRecordEnterpriseID, directoryRecordID, "--body", "{"},
	} {
		assertDirectoryRecordUsage(t, args)
	}

	file := filepath.Join(t.TempDir(), "directory-record.json")
	if err := os.WriteFile(file, []byte(directoryRecordBody), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, row := range []directoryRecordFixture{directoryRecordFixtures()[1], directoryRecordFixtures()[3]} {
		for _, body := range []string{directoryRecordBody, "@" + file, "-"} {
			args := append([]string(nil), row.args[:len(row.args)-1]...)
			input := io.Reader(strings.NewReader(""))
			if body == "-" {
				input = strings.NewReader(directoryRecordBody)
			}
			args[len(args)-1] = body
			assertDirectoryRecordRequest(t, args, input, row.method, row.path, directoryRecordBody)
		}
	}
}

func TestDirectoryRecordDeleteDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	command := configuredDirectoryRecordCommand(t, server.URL)
	command.SetIn(strings.NewReader("no\n"))
	command.SetArgs([]string{"directory-record", "delete", directoryRecordEnterpriseID, directoryRecordID})
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("delete refusal error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("declined requests = %d, want 0", requests)
	}
}

func executeDirectoryRecordFixture(t *testing.T, row directoryRecordFixture, apiError bool) {
	t.Helper()
	requests := 0
	fixture, status := row.name+"-success.json", row.status
	arguments := append([]string(nil), row.args...)
	if apiError {
		fixture, status = row.name+"-api-error.json", row.errorStatus
		if row.all {
			arguments = append(arguments[:len(arguments)-2], "--json")
		}
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != row.method || request.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, row.method, row.path)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" || request.Header.Get("Accept") != "application/json" {
			t.Errorf("authorization = %q, accept = %q", request.Header.Get("Authorization"), request.Header.Get("Accept"))
		}
		if row.all && !apiError && requests == 2 {
			if got, want := request.URL.Query().Encode(), (url.Values{"offset": {"1"}}).Encode(); got != want {
				t.Errorf("second-page query = %q, want %q", got, want)
			}
			fixture = row.name + "-second-page.json"
		} else if got, want := request.URL.Query().Encode(), row.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil || string(body) != row.body {
			t.Errorf("body = %q, error = %v, want %q", body, err, row.body)
		}
		if row.body != "" && request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q, want application/json", request.Header.Get("Content-Type"))
		}
		response := readDirectoryRecordFixture(t, fixture)
		if row.all && !apiError && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?offset=1"), 1)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(response)
	}))
	defer server.Close()

	command := configuredDirectoryRecordCommand(t, server.URL)
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs(arguments)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != status || esperruntime.ExitCode(err) != 1 || output.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("API error = %v, stdout = %q, stderr = %q", err, output.String(), stderr.String())
		}
		if want := readDirectoryRecordFixture(t, fixture); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %q, want %q", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if row.all && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if want := readDirectoryRecordFixture(t, row.name+"-success.golden"); !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.Bytes(), want)
	}
}

func assertDirectoryRecordUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}

func assertDirectoryRecordRequest(t *testing.T, args []string, input io.Reader, method, path, body string) {
	t.Helper()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != method || request.URL.Path != path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, method, path)
		}
		got, err := io.ReadAll(request.Body)
		if err != nil || string(got) != body {
			t.Errorf("body = %q, error = %v, want %q", got, err, body)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(readDirectoryRecordFixture(t, "v1-directory-record-get-success.json"))
	}))
	defer server.Close()
	command := configuredDirectoryRecordCommand(t, server.URL)
	command.SetIn(input)
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute(%v): %v", args, err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func configuredDirectoryRecordCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func directoryRecordValidateFixture(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readDirectoryRecordFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

func readDirectoryRecordFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "directory-record", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
