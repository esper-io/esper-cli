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
	foundryBuildID       = "11111111-1111-4111-8111-111111111111"
	foundryDeviceModelID = "22222222-2222-4222-8222-222222222222"
	foundryLockedBuildID = "33333333-3333-4333-8333-333333333333"
	foundryBuildBody     = `{"approved":true,"user_remarks":"Approved for rollout"}`
	foundryDeviceBody    = `{"auto_update":false,"locked_build_id":"33333333-3333-4333-8333-333333333333"}`
)

type foundryFixture struct {
	key, name, method, path, body string
	args                          []string
	query                         url.Values
	status, errorStatus           int
	pagination                    bool
}

func foundryFixtures() []foundryFixture {
	return []foundryFixture{
		{"foundry GET /v1/foundry/builds/", "foundry-build-list", http.MethodGet, "/v1/foundry/builds/", "", []string{"foundry-build", "list", "--foundation-major-number", "15", "--android-major-version", "14", "--approved", "true", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"foundation_major_number": {"15"}, "android_major_version": {"14"}, "approved": {"true"}, "limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusBadRequest, true},
		{"foundry GET /v1/foundry/builds/{build_id}/", "foundry-build-get", http.MethodGet, "/v1/foundry/builds/" + foundryBuildID + "/", "", []string{"foundry-build", "get", foundryBuildID, "--json"}, nil, http.StatusOK, http.StatusNotFound, false},
		{"foundry PUT /v1/foundry/builds/{build_id}/", "foundry-build-update", http.MethodPut, "/v1/foundry/builds/" + foundryBuildID + "/", foundryBuildBody, []string{"foundry-build", "update", foundryBuildID, "--approved", "true", "--user-remarks", "Approved for rollout", "--json"}, nil, http.StatusOK, http.StatusBadRequest, false},
		{"foundry GET /v1/foundry/device-models/", "foundry-device-model-list", http.MethodGet, "/v1/foundry/device-models/", "", []string{"foundry-device-model", "list", "--manufacturer", "Acme", "--update-compatible-android-api-level-begin", "30", "--update-compatible-android-api-level-end", "35", "--device-model-id", foundryDeviceModelID, "--device-android-api-level", "34", "--device-display-name", "Acme Kiosk", "--device-foundation-major-number", "15", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"manufacturer": {"Acme"}, "update_compatible_android_api_level_begin": {"30"}, "update_compatible_android_api_level_end": {"35"}, "device_model_id": {foundryDeviceModelID}, "device_android_api_level": {"34"}, "device_display_name": {"Acme Kiosk"}, "device_foundation_major_number": {"15"}, "limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusBadRequest, true},
		{"foundry PUT /v1/foundry/device-models/{device_model_id}/", "foundry-device-model-update", http.MethodPut, "/v1/foundry/device-models/" + foundryDeviceModelID + "/", foundryDeviceBody, []string{"foundry-device-model", "update", foundryDeviceModelID, "--auto-update", "false", "--locked-build-id", foundryLockedBuildID, "--json"}, nil, http.StatusOK, http.StatusBadRequest, false},
		{"foundry GET /v1/foundry/events/", "foundry-event-list", http.MethodGet, "/v1/foundry/events/", "", []string{"foundry-event", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusBadRequest, true},
	}
}

func TestFoundryOperationCoverage(t *testing.T) {
	want, got := map[string]bool{}, map[string]bool{}
	for _, row := range foundryFixtures() {
		if want[row.key] {
			t.Fatalf("duplicate fixture row %s", row.key)
		}
		want[row.key] = true
	}
	if len(want) != 6 {
		t.Fatalf("fixture rows = %d, want 6", len(want))
	}
	for _, operation := range generated.Operations() {
		if strings.HasPrefix(operation.Noun, "foundry-") {
			got[operation.Generation+" "+operation.Method+" "+operation.Path] = true
			if operation.Destructive {
				t.Fatalf("%s is unexpectedly destructive", operation.Noun)
			}
		}
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(want), len(got))
	}
}

func TestFoundryFixturesMatchResponseContracts(t *testing.T) {
	document := readFoundryDocument(t)
	for _, row := range foundryFixtures() {
		parts := strings.SplitN(row.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		foundryValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		if row.pagination {
			foundryValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-second-page.json")
		}
		value := foundryValidateFixture(t, document, fixtureSchemaResponseForStatus(document, operation, row.errorStatus), row.name+"-api-error.json")
		if object, ok := value.(map[string]any); !ok || object["code"] != float64(row.errorStatus) {
			t.Fatalf("%s API error code = %#v, want %d", row.name, value, row.errorStatus)
		}
	}
}

func TestFoundryGoldenFixtures(t *testing.T) {
	for _, row := range foundryFixtures() {
		t.Run(row.name, func(t *testing.T) { executeFoundryFixture(t, row, false) })
	}
}

func TestFoundryAPIErrors(t *testing.T) {
	for _, row := range foundryFixtures() {
		t.Run(row.name, func(t *testing.T) { executeFoundryFixture(t, row, true) })
	}
}

func TestFoundryPaginationMerges(t *testing.T) {
	for _, row := range foundryFixtures() {
		if !row.pagination {
			continue
		}
		t.Run(row.name, func(t *testing.T) {
			first, second := readFoundryFixture(t, row.name+"-success.json"), readFoundryFixture(t, row.name+"-second-page.json")
			one, err := esperruntime.UnwrapAppsEnvelope(first)
			if err != nil || one.Next == "" || one.Previous != "" {
				t.Fatalf("first page = %#v, error = %v", one, err)
			}
			two, err := esperruntime.UnwrapAppsEnvelope(second)
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
			if err != nil || !bytes.Equal(formatted, readFoundryFixture(t, row.name+"-success.golden")) {
				t.Fatalf("merged = %s, error = %v", merged, err)
			}
		})
	}
}

func TestFoundryInputRules(t *testing.T) {
	root := NewRootCommand()
	for _, commandPath := range [][]string{{"foundry-build", "list"}, {"foundry-build", "get"}, {"foundry-build", "update"}, {"foundry-device-model", "list"}, {"foundry-device-model", "update"}, {"foundry-event", "list"}} {
		command, _, err := root.Find(commandPath)
		if err != nil {
			t.Fatal(err)
		}
		if command.Flags().Lookup("authorization") != nil {
			t.Fatalf("%s exposes internal authorization flag", strings.Join(commandPath, " "))
		}
	}

	for _, args := range [][]string{
		{"foundry-build", "get"},
		{"foundry-build", "update", foundryBuildID},
		{"foundry-build", "update", foundryBuildID, "--user-remarks", "Approved for rollout"},
		{"foundry-build", "update", foundryBuildID, "--body", "{"},
		{"foundry-build", "update", foundryBuildID, "--body", foundryBuildBody, "--approved", "true"},
		{"foundry-device-model", "update", foundryDeviceModelID},
		{"foundry-device-model", "update", foundryDeviceModelID, "--body", "{"},
		{"foundry-device-model", "update", foundryDeviceModelID, "--body", foundryDeviceBody, "--auto-update", "false"},
	} {
		assertFoundryUsage(t, args)
	}

	for _, row := range []foundryFixture{foundryFixtures()[2], foundryFixtures()[4]} {
		file := filepath.Join(t.TempDir(), row.name+".json")
		if err := os.WriteFile(file, []byte(row.body), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, body := range []string{row.body, "@" + file, "-"} {
			input := io.Reader(strings.NewReader(""))
			if body == "-" {
				input = strings.NewReader(row.body)
			}
			assertFoundryRequest(t, []string{row.args[0], row.args[1], row.args[2], "--body", body}, input, row)
		}
	}
}

func executeFoundryFixture(t *testing.T, row foundryFixture, apiError bool) {
	t.Helper()
	arguments := append([]string(nil), row.args...)
	if apiError && row.pagination {
		arguments = append(arguments[:len(arguments)-2], "--json")
	}
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != row.method || request.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, row.method, row.path)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" || request.Header.Get("Accept") != "application/json" {
			t.Errorf("authorization = %q, accept = %q", request.Header.Get("Authorization"), request.Header.Get("Accept"))
		}
		if row.pagination && !apiError && requests == 2 {
			if got, want := request.URL.Query().Encode(), foundrySecondPageQuery(row).Encode(); got != want {
				t.Errorf("second page query = %q, want %q", got, want)
			}
		} else if got, want := request.URL.Query().Encode(), row.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		assertFoundryBody(t, request, row.body)
		name, status := row.name+"-success.json", row.status
		if apiError {
			name, status = row.name+"-api-error.json", row.errorStatus
		} else if row.pagination && requests == 2 {
			name = row.name + "-second-page.json"
		}
		response := readFoundryFixture(t, name)
		if row.pagination && !apiError && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?"+foundrySecondPageQuery(row).Encode()), 1)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(response)
	}))
	defer server.Close()

	command := configuredFoundryCommand(t, server.URL)
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs(arguments)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != row.errorStatus || esperruntime.ExitCode(err) != 1 || output.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("API error = %v, stdout = %q, stderr = %q", err, output.String(), stderr.String())
		}
		if want := readFoundryFixture(t, row.name+"-api-error.json"); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %q, want %q", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if row.pagination && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if stderr.Len() != 0 || !bytes.Equal(output.Bytes(), readFoundryFixture(t, row.name+"-success.golden")) {
		t.Fatalf("stdout = %q, stderr = %q", output.Bytes(), stderr.Bytes())
	}
}

func foundrySecondPageQuery(row foundryFixture) url.Values {
	query := make(url.Values, len(row.query))
	for name, values := range row.query {
		query[name] = append([]string(nil), values...)
	}
	query.Set("offset", "1")
	return query
}

func assertFoundryBody(t *testing.T, request *http.Request, want string) {
	t.Helper()
	got, err := io.ReadAll(request.Body)
	if err != nil || string(got) != want {
		t.Errorf("body = %q, error = %v, want %q", got, err, want)
	}
	if want != "" && request.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type = %q, want application/json", request.Header.Get("Content-Type"))
	}
}

func assertFoundryRequest(t *testing.T, args []string, input io.Reader, row foundryFixture) {
	t.Helper()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != row.method || request.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, row.method, row.path)
		}
		assertFoundryBody(t, request, row.body)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(row.status)
		_, _ = writer.Write(readFoundryFixture(t, row.name+"-success.json"))
	}))
	defer server.Close()
	command := configuredFoundryCommand(t, server.URL)
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

func assertFoundryUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}

func configuredFoundryCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func readFoundryDocument(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "foundry.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func foundryValidateFixture(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readFoundryFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

func readFoundryFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "foundry", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
