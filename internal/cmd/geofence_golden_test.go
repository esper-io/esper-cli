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
	geofenceEnterpriseID = "11111111-1111-4111-8111-111111111111"
	geofenceID           = "22222222-2222-4222-8222-222222222222"
	geofenceCreateBody   = `{"name":"Warehouse perimeter","description":"Main distribution center","latitude":"37.7749","longitude":"-122.4194","devices":["44444444-4444-4444-8444-444444444444"],"radius":0.25,"radius_unit":"KILOMETERS","device_actions":["LOCK_DOWN","BEEP"]}`
	geofenceUpdateBody   = `{"latitude":"37.7749","longitude":"-122.4194","name":"Warehouse perimeter","radius":0.25,"radius_unit":"KILOMETERS"}`
	geofencePatchBody    = `{"radius":0.5,"radius_unit":"KILOMETERS"}`
)

type geofenceFixture struct {
	key, name, method, path, body string
	args                          []string
	query                         url.Values
	status, errorStatus           int
	all, destructive              bool
}

type geofenceOperationMetadata struct {
	Generation, Method, Path, Noun, Verb, Pagination, ScopeParent, Summary, AliasOf, SuccessMedia string
	Command                                                                                       []string
	Destructive                                                                                   bool
}

func geofenceFixtures() []geofenceFixture {
	return []geofenceFixture{
		{"v0 GET /v0/enterprise/{enterprise_id}/geofence/", "v0-geofence-list", http.MethodGet, "/v0/enterprise/" + geofenceEnterpriseID + "/geofence/", "", []string{"api", "v0", "geofence", "list", "--enterprise", geofenceEnterpriseID, "--search", "warehouse", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"search": {"warehouse"}, "limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusUnauthorized, true, false},
		{"v0 GET /v0/enterprise/{enterprise_id}/geofence/{geofence_id}/", "v0-geofence-get", http.MethodGet, "/v0/enterprise/" + geofenceEnterpriseID + "/geofence/" + geofenceID + "/", "", []string{"geofence", "get", geofenceEnterpriseID, geofenceID, "--json"}, nil, http.StatusOK, http.StatusNotFound, false, false},
		{"v0 PUT /v0/enterprise/{enterprise_id}/geofence/{geofence_id}/", "v0-geofence-update", http.MethodPut, "/v0/enterprise/" + geofenceEnterpriseID + "/geofence/" + geofenceID + "/", geofenceUpdateBody, []string{"geofence", "update", geofenceEnterpriseID, geofenceID, "--name", "Warehouse perimeter", "--latitude", "37.7749", "--longitude", "-122.4194", "--radius", "0.25", "--radius-unit", "KILOMETERS", "--json"}, nil, http.StatusOK, http.StatusBadRequest, false, false},
		{"v0 PATCH /v0/enterprise/{enterprise_id}/geofence/{geofence_id}/", "v0-geofence-partial-update", http.MethodPatch, "/v0/enterprise/" + geofenceEnterpriseID + "/geofence/" + geofenceID + "/", geofencePatchBody, []string{"geofence", "partial-update", geofenceEnterpriseID, geofenceID, "--radius", "0.5", "--radius-unit", "KILOMETERS", "--json"}, nil, http.StatusOK, http.StatusBadRequest, false, false},
		{"v0 DELETE /v0/enterprise/{enterprise_id}/geofence/{geofence_id}/", "v0-geofence-delete", http.MethodDelete, "/v0/enterprise/" + geofenceEnterpriseID + "/geofence/" + geofenceID + "/", "", []string{"geofence", "delete", geofenceEnterpriseID, geofenceID, "--yes"}, nil, http.StatusNoContent, http.StatusNotFound, false, true},
		{"v0 POST /v0/enterprise/{enterprise_id}/create-apply-geo-fence/", "v0-geofence-create", http.MethodPost, "/v0/enterprise/" + geofenceEnterpriseID + "/create-apply-geo-fence/", geofenceCreateBody, []string{"api", "v0", "geofence", "create", "--enterprise", geofenceEnterpriseID, "--body", geofenceCreateBody, "--json"}, nil, http.StatusCreated, http.StatusBadRequest, false, false},
	}
}

func TestGeofenceOperationCoverage(t *testing.T) {
	want := map[string]geofenceOperationMetadata{
		"getAllGeofences":            {"v0", http.MethodGet, "/v0/enterprise/{enterprise_id}/geofence/", "geofence", "list", "limit-offset", "enterprise", "List Geofences in Enterprise", "", "application/json", []string{"api", "v0", "geofence", "list"}, false},
		"getGeofence":                {"v0", http.MethodGet, "/v0/enterprise/{enterprise_id}/geofence/{geofence_id}/", "geofence", "get", "none", "", "Get geofence information", "", "application/json", []string{"geofence", "get"}, false},
		"updateGeofence":             {"v0", http.MethodPut, "/v0/enterprise/{enterprise_id}/geofence/{geofence_id}/", "geofence", "update", "none", "", "Update geofence information", "", "application/json", []string{"geofence", "update"}, false},
		"partialUpdateGeofence":      {"v0", http.MethodPatch, "/v0/enterprise/{enterprise_id}/geofence/{geofence_id}/", "geofence", "partial-update", "none", "", "Partially updates geofence information", "", "application/json", []string{"geofence", "partial-update"}, false},
		"deleteGeofence":             {"v0", http.MethodDelete, "/v0/enterprise/{enterprise_id}/geofence/{geofence_id}/", "geofence", "delete", "none", "", "Delete a geofence", "", "", []string{"geofence", "delete"}, true},
		"createGeofence":             {"v0", http.MethodPost, "/v0/enterprise/{enterprise_id}/create-apply-geo-fence/", "geofence", "create", "none", "enterprise", "Create a geofence", "", "application/json", []string{"api", "v0", "geofence", "create"}, false},
		"getGeofenceList":            {"v0", http.MethodGet, "/v0/enterprise/{enterprise_id}/create-apply-geo-fence/", "geofence", "list", "limit-offset", "enterprise", "List Geofences in Enterprise", "getAllGeofences", "application/json", []string{"api", "v0", "geofence", "list"}, false},
		"getTheGeofence":             {"v0", http.MethodGet, "/v0/enterprise/{enterprise_id}/create-apply-geofence/{geofence_id}/", "geofence", "get", "none", "", "Get geofence information", "getGeofence", "application/json", []string{"geofence", "get"}, false},
		"deleteTheGeofence":          {"v0", http.MethodDelete, "/v0/enterprise/{enterprise_id}/create-apply-geofence/{geofence_id}/", "geofence", "delete", "none", "", "Delete a geofence", "deleteGeofence", "", []string{"geofence", "delete"}, true},
		"geofence_createGeofence":    {"v1", http.MethodPost, "/geofence/v1/geofences", "geofence", "create", "none", "", "Create a new geofence", "", "application/json", []string{"geofence", "create"}, false},
		"geofence_listGeofences":     {"v1", http.MethodGet, "/geofence/v1/geofences", "geofence", "list", "none", "", "List all geofences", "", "application/json", []string{"geofence", "list"}, false},
		"geofence_getDeviceSummary":  {"v1", http.MethodGet, "/geofence/v1/geofences/{geofence_id}/device-summary", "geofence-device-summary", "get", "none", "", "Get device summary for a geofence", "", "application/json", []string{"geofence-device-summary", "get"}, false},
		"geofence_getBlueprintUsage": {"v1", http.MethodGet, "/geofence/v1/geofences/{geofence_id}/blueprints", "geofence-blueprint", "list", "none", "geofence", "Get blueprint usage statistics for a geofence", "", "application/json", []string{"geofence-blueprint", "list"}, false},
		"geofence_getDevices":        {"v1", http.MethodGet, "/geofence/v1/geofences/{geofence_id}/devices", "geofence-device", "list", "none", "geofence", "Get devices for a geofence", "", "application/json", []string{"geofence-device", "list"}, false},
	}
	got := map[string]geofenceOperationMetadata{}
	for _, operation := range generated.Operations() {
		if operation.Noun == "geofence" || strings.HasPrefix(operation.Noun, "geofence-") {
			got[operation.OperationID] = geofenceOperationMetadata{operation.Generation, operation.Method, operation.Path, operation.Noun, operation.Verb, operation.Pagination, operation.ScopeParent, operation.Summary, operation.AliasOf, operation.SuccessMedia, operation.Command, operation.Destructive}
		}
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("generated geofence inventory = %#v, want %#v", got, want)
	}
	for _, noun := range []string{"geo-fence", "geo-fence-create", "create-apply-geo-fence", "create-apply-geofence", "the-geofence"} {
		if command, _, err := NewRootCommand().Find([]string{noun}); err == nil && command.Use == noun {
			t.Fatalf("malformed noun command %q exists", noun)
		}
	}
}

func TestGeofenceFixtureInventory(t *testing.T) {
	want := map[string]bool{"README.md": true, "RERECORD_WITH_TENANT": true}
	for _, row := range geofenceFixtures() {
		want[row.name+"-success.json"] = true
		want[row.name+"-success.golden"] = true
		want[row.name+"-api-error.json"] = true
		if row.all {
			want[row.name+"-second-page.json"] = true
		}
	}
	entries, err := os.ReadDir(filepath.Join("..", "..", "spec", "fixtures", "geofence"))
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]bool, len(entries))
	for _, entry := range entries {
		got[entry.Name()] = true
	}
	if len(want) != 21 || !reflect.DeepEqual(want, got) {
		t.Fatalf("geofence fixture inventory = %#v, want 21 exact files", got)
	}
}

func TestGeofenceFixturesMatchResponseContracts(t *testing.T) {
	document := readGeofenceDocument(t)
	for _, row := range geofenceFixtures() {
		parts := strings.SplitN(row.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		if row.status == http.StatusNoContent {
			if data := readGeofenceFixture(t, row.name+"-success.json"); len(data) != 0 {
				t.Fatalf("%s 204 success fixture = %q, want empty", row.name, data)
			}
		} else {
			geofenceValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		}
		if row.all {
			geofenceValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-second-page.json")
		}
		value := geofenceValidateFixture(t, document, fixtureSchemaResponseForStatus(document, operation, row.errorStatus), row.name+"-api-error.json")
		if object, ok := value.(map[string]any); !ok || object["status"] != float64(row.errorStatus) {
			t.Fatalf("%s API error = %#v, want status %d", row.name, value, row.errorStatus)
		}
	}
}

func TestGeofenceGoldenFixtures(t *testing.T) {
	for _, row := range geofenceFixtures() {
		t.Run(row.name, func(t *testing.T) { executeGeofenceFixture(t, row, false) })
	}
}

func TestGeofenceAPIErrors(t *testing.T) {
	for _, row := range geofenceFixtures() {
		t.Run(row.name, func(t *testing.T) { executeGeofenceFixture(t, row, true) })
	}
}

func TestGeofencePaginationMerges(t *testing.T) {
	first, second := readGeofenceFixture(t, "v0-geofence-list-success.json"), readGeofenceFixture(t, "v0-geofence-list-second-page.json")
	one, err := esperruntime.UnwrapLimitOffset(first)
	if err != nil || one.Next == "" || one.Previous != "" {
		t.Fatalf("first page = %#v, error = %v", one, err)
	}
	two, err := esperruntime.UnwrapLimitOffset(second)
	if err != nil || two.Next != "" || two.Previous == "" {
		t.Fatalf("second page = %#v, error = %v", two, err)
	}
	merged, err := esperruntime.MarshalMergedResults(append(one.Results, two.Results...))
	var value any
	var formatted []byte
	if err == nil {
		err = json.Unmarshal(merged, &value)
	}
	if err == nil {
		formatted, err = json.MarshalIndent(value, "", "  ")
		formatted = append(formatted, '\n')
	}
	if err != nil || !bytes.Equal(formatted, readGeofenceFixture(t, "v0-geofence-list-success.golden")) {
		t.Fatalf("merged = %s, error = %v", merged, err)
	}
}

func TestGeofenceInputRules(t *testing.T) {
	root := NewRootCommand()
	for _, path := range [][]string{{"api", "v0", "geofence", "list"}, {"api", "v0", "geofence", "create"}} {
		command, _, err := root.Find(path)
		if err != nil || command.Flags().Lookup("enterprise") == nil {
			t.Fatalf("%s enterprise scope flag = %v, error = %v", strings.Join(path, " "), command.Flags().Lookup("enterprise"), err)
		}
	}
	for _, path := range [][]string{{"geofence", "get"}, {"geofence", "update"}, {"geofence", "partial-update"}, {"geofence", "delete"}} {
		command, _, err := root.Find(path)
		if err != nil || command.Flags().Lookup("enterprise") != nil || command.Flags().Lookup("geofence-id") != nil {
			t.Fatalf("%s item path flags = enterprise:%v geofence-id:%v error:%v", strings.Join(path, " "), command.Flags().Lookup("enterprise"), command.Flags().Lookup("geofence-id"), err)
		}
	}
	for _, path := range [][]string{{"api", "v0", "geofence", "create"}, {"geofence", "update"}, {"geofence", "partial-update"}} {
		command, _, err := root.Find(path)
		if err != nil || command.Flags().Lookup("body") == nil || command.Flags().Lookup("authorization") != nil || command.Flags().Lookup("output") != nil {
			t.Fatalf("%s flags do not expose the expected JSON input surface", strings.Join(path, " "))
		}
	}
	create, _, _ := root.Find([]string{"api", "v0", "geofence", "create"})
	for _, name := range []string{"name", "latitude", "longitude", "devices", "radius", "radius-unit", "device-actions"} {
		if create.Flags().Lookup(name) != nil {
			t.Fatalf("geofence create unexpectedly exposes --%s", name)
		}
	}
	for _, args := range [][]string{
		{"api", "v0", "geofence", "list"},
		{"api", "v0", "geofence", "create", "--enterprise", geofenceEnterpriseID},
		{"api", "v0", "geofence", "create", "--enterprise", geofenceEnterpriseID, "--body", "{"},
		{"geofence", "update", geofenceEnterpriseID, geofenceID},
		{"geofence", "update", geofenceEnterpriseID, geofenceID, "--name", "Warehouse perimeter", "--latitude", "37.7749"},
		{"geofence", "update", geofenceEnterpriseID, geofenceID, "--name", "Warehouse perimeter", "--longitude", "-122.4194"},
		{"geofence", "update", geofenceEnterpriseID, geofenceID, "--latitude", "37.7749", "--longitude", "-122.4194"},
		{"geofence", "update", geofenceEnterpriseID, geofenceID, "--body", "{"},
		{"geofence", "update", geofenceEnterpriseID, geofenceID, "--body", geofenceUpdateBody, "--name", "Warehouse perimeter"},
		{"geofence", "partial-update", geofenceEnterpriseID, geofenceID},
		{"geofence", "partial-update", geofenceEnterpriseID, geofenceID, "--body", "{"},
		{"geofence", "partial-update", geofenceEnterpriseID, geofenceID, "--body", geofencePatchBody, "--radius", "0.5"},
	} {
		assertGeofenceUsage(t, args)
	}

	for _, row := range []geofenceFixture{geofenceFixtures()[2], geofenceFixtures()[3]} {
		file := filepath.Join(t.TempDir(), row.name+".json")
		if err := os.WriteFile(file, []byte(row.body), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, body := range []string{row.body, "@" + file, "-"} {
			input := io.Reader(strings.NewReader(""))
			if body == "-" {
				input = strings.NewReader(row.body)
			}
			assertGeofenceRequest(t, []string{row.args[0], row.args[1], row.args[2], row.args[3], "--body", body}, input, row)
		}
	}
	file := filepath.Join(t.TempDir(), "geofence-create.json")
	if err := os.WriteFile(file, []byte(geofenceCreateBody), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{geofenceCreateBody, "@" + file, "-"} {
		input := io.Reader(strings.NewReader(""))
		if body == "-" {
			input = strings.NewReader(geofenceCreateBody)
		}
		assertGeofenceRequest(t, []string{"api", "v0", "geofence", "create", "--enterprise", geofenceEnterpriseID, "--body", body}, input, geofenceFixtures()[5])
	}
}

func TestGeofenceDeleteDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	command := configuredGeofenceCommand(t, server.URL)
	command.SetIn(strings.NewReader("no\n"))
	command.SetArgs([]string{"geofence", "delete", geofenceEnterpriseID, geofenceID})
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("delete refusal error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("declined requests = %d, want 0", requests)
	}
}

func executeGeofenceFixture(t *testing.T, row geofenceFixture, apiError bool) {
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
			if got, want := request.URL.Query().Encode(), geofenceSecondPageQuery(row).Encode(); got != want {
				t.Errorf("second page query = %q, want %q", got, want)
			}
			fixture = row.name + "-second-page.json"
		} else if got, want := request.URL.Query().Encode(), row.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		assertGeofenceBody(t, request, row.body)
		response := readGeofenceFixture(t, fixture)
		if row.all && !apiError && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?"+geofenceSecondPageQuery(row).Encode()), 1)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(response)
	}))
	defer server.Close()

	command := configuredGeofenceCommand(t, server.URL)
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
		if want := readGeofenceFixture(t, fixture); !bytes.Equal(value.Body, want) {
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
	if stderr.Len() != 0 || !bytes.Equal(output.Bytes(), readGeofenceFixture(t, row.name+"-success.golden")) {
		t.Fatalf("stdout = %q, stderr = %q", output.Bytes(), stderr.Bytes())
	}
}

func geofenceSecondPageQuery(row geofenceFixture) url.Values {
	query := make(url.Values, len(row.query))
	for name, values := range row.query {
		query[name] = append([]string(nil), values...)
	}
	query.Set("offset", "1")
	return query
}

func assertGeofenceBody(t *testing.T, request *http.Request, want string) {
	t.Helper()
	got, err := io.ReadAll(request.Body)
	if err != nil || string(got) != want {
		t.Errorf("body = %q, error = %v, want %q", got, err, want)
	}
	if want != "" && request.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type = %q, want application/json", request.Header.Get("Content-Type"))
	}
}

func assertGeofenceRequest(t *testing.T, args []string, input io.Reader, row geofenceFixture) {
	t.Helper()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != row.method || request.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, row.method, row.path)
		}
		assertGeofenceBody(t, request, row.body)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(row.status)
		_, _ = writer.Write(readGeofenceFixture(t, row.name+"-success.json"))
	}))
	defer server.Close()
	command := configuredGeofenceCommand(t, server.URL)
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

func assertGeofenceUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}

func configuredGeofenceCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func readGeofenceDocument(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "v0.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func geofenceValidateFixture(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readGeofenceFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

func readGeofenceFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "geofence", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
