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

type deviceSupportFixture struct {
	key, name, method, path, body string
	args                          []string
	query                         url.Values
	status, errorStatus           int
	pagination                    string
	destructive                   bool
}

func deviceSupportFixtures() []deviceSupportFixture {
	return []deviceSupportFixture{
		{"legacy GET /enterprise/{enterprise_id}/device/{device_id}/download/eventfeed/", "legacy-device-eventfeed-list", http.MethodGet, "/enterprise/tenant-1/device/device-1/download/eventfeed/", "", []string{"device-eventfeed", "list", "--enterprise", "tenant-1", "--device", "device-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, "limit-offset", false},
		{"v0 GET /device/v0/devices/{id}/", "v0-device-request-get", http.MethodGet, "/device/v0/devices/device-1/", "", []string{"device-request", "get", "device-1", "--json"}, nil, 200, 404, "", false},
		{"v0 DELETE /device/v0/devices/{id}/", "v0-device-request-delete", http.MethodDelete, "/device/v0/devices/device-1/", "", []string{"device-request", "delete", "device-1", "--yes", "--json"}, nil, 200, 404, "", true},
		{"v0 GET /device/v0/devices/{id}/devicestate", "v0-devicestate-get", http.MethodGet, "/device/v0/devices/device-1/devicestate", "", []string{"devicestate", "get", "device-1", "--json"}, nil, 200, 404, "", false},
		{"v0 GET /device/v0/heartbeat/{id}/", "v0-device-heartbeat-get", http.MethodGet, "/device/v0/heartbeat/device-1/", "", []string{"device-heartbeat", "get", "device-1", "--json"}, nil, 200, 404, "", false},
		{"v2 GET /v2/heartbeat/", "v2-device-heartbeat-list", http.MethodGet, "/v2/heartbeat/", "", []string{"device-heartbeat-list", "list", "--last-seen-gt", "2026-08-24T00:00:00Z", "--last-seen-lt", "2026-08-25T00:00:00Z", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"last_seen_gt": {"2026-08-24T00:00:00Z"}, "last_seen_lt": {"2026-08-25T00:00:00Z"}, "limit": {"1"}, "offset": {"0"}}, 200, 400, "apps-envelope", false},
		{"v2 GET /v2/devices/{deviceId}/google-accounts/policy/", "v2-device-google-account-policy-get", http.MethodGet, "/v2/devices/device-1/google-accounts/policy/", "", []string{"device-google-account-policy", "get", "device-1", "--json"}, nil, 200, 404, "", false},
		{"v2 PUT /v2/devices/{deviceId}/google-accounts/policy/", "v2-device-google-account-policy-update", http.MethodPut, "/v2/devices/device-1/google-accounts/policy/", `{"applications":[{"product_id":"app:com.example.fixture","app_action":"install"}]}`, []string{"device-google-account-policy", "update", "device-1", "--body", `{"applications":[{"product_id":"app:com.example.fixture","app_action":"install"}]}`, "--json"}, nil, 200, 400, "", false},
		{"v2 GET /v2/devices/{deviceId}/google-accounts/emm-managed/", "v2-device-google-account-emm-managed-get", http.MethodGet, "/v2/devices/device-1/google-accounts/emm-managed/", "", []string{"device-google-account-emm-managed", "get", "device-1", "--json"}, nil, 200, 404, "", false},
		{"v2 GET /v2/devices/{deviceId}/google-accounts/", "v2-google-account-list", http.MethodGet, "/v2/devices/device-1/google-accounts/", "", []string{"google-account", "list", "--device", "device-1", "--all", "--json"}, nil, 200, 400, "apps-envelope", false},
		{"v2 PUT /v2/devices/{deviceId}/google-accounts/", "v2-google-account-update", http.MethodPut, "/v2/devices/device-1/google-accounts/", `{"google_device_id":"google-device-1","google_user_id":"google-user-1"}`, []string{"google-account", "update", "device-1", "--google-device-id", "google-device-1", "--google-user-id", "google-user-1", "--json"}, nil, 200, 400, "", false},
		{"v2 POST /v2/devices/{deviceId}/google-accounts/", "v2-google-account-create", http.MethodPost, "/v2/devices/device-1/google-accounts/", "", []string{"google-account", "create", "--device", "device-1", "--json"}, nil, 200, 409, "", false},
		{"v2 DELETE /v2/devices/{deviceId}/google-accounts/", "v2-google-account-delete", http.MethodDelete, "/v2/devices/device-1/google-accounts/", "", []string{"google-account", "delete", "device-1", "--yes", "--json"}, nil, 200, 404, "", true},
		{"v2 GET /v2/foundationversions", "v2-foundation-version-list", http.MethodGet, "/v2/foundationversions", "", []string{"foundation-version-list", "list", "--search", "1.0", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"search": {"1.0"}, "limit": {"1"}, "offset": {"0"}}, 200, 400, "limit-offset", false},
		{"v2 GET /v2/rv-activity-feed/", "v2-rv-activity-feed-list", http.MethodGet, "/v2/rv-activity-feed/", "", []string{"rv-activity-feed", "list", "--states", "TERMINATED", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"states": {"TERMINATED"}, "limit": {"1"}, "offset": {"0"}}, 200, 400, "apps-envelope", false},
	}
}

func TestDeviceSupportOperationCoverage(t *testing.T) {
	want := map[string]bool{}
	for _, row := range deviceSupportFixtures() {
		if want[row.key] {
			t.Fatalf("duplicate fixture row %s", row.key)
		}
		want[row.key] = true
	}
	if len(want) != 15 {
		t.Fatalf("fixture rows = %d, want 15", len(want))
	}
	nouns := map[string]bool{"device-eventfeed": true, "device-google-account-emm-managed": true, "device-google-account-policy": true, "device-heartbeat": true, "device-heartbeat-list": true, "device-request": true, "devicestate": true, "foundation-version-list": true, "google-account": true, "rv-activity-feed": true}
	got := map[string]bool{}
	for _, operation := range generated.Operations() {
		if nouns[operation.Noun] {
			got[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if len(got) != 15 || !reflect.DeepEqual(want, got) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(want), len(got))
	}
}

func TestDeviceSupportFixturesMatchResponseContracts(t *testing.T) {
	documents := map[string]map[string]any{}
	for _, file := range []string{"legacy.yaml", "v0.yaml", "v2.yaml"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", file))
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		documents[strings.TrimSuffix(file, ".yaml")] = document
	}
	for _, row := range deviceSupportFixtures() {
		parts := strings.SplitN(row.key, " ", 3)
		document := documents[parts[0]]
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		deviceSupportValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		if row.pagination != "" {
			deviceSupportValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-second-page.json")
		}
		deviceSupportValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.errorStatus), row.name+"-api-error.json")
	}
}

func TestDeviceSupportGoldenFixtures(t *testing.T) {
	for _, row := range deviceSupportFixtures() {
		t.Run(row.name, func(t *testing.T) { executeDeviceSupportFixture(t, row, false) })
	}
}

func TestDeviceSupportAPIErrors(t *testing.T) {
	for _, row := range deviceSupportFixtures() {
		t.Run(row.name, func(t *testing.T) { executeDeviceSupportFixture(t, row, true) })
	}
}

func TestDeviceSupportPaginationFixtures(t *testing.T) {
	for _, row := range deviceSupportFixtures() {
		if row.pagination == "" {
			continue
		}
		t.Run(row.name, func(t *testing.T) {
			first := readDeviceSupportFixture(t, row.name+"-success.json")
			second := readDeviceSupportFixture(t, row.name+"-second-page.json")
			var one, two esperruntime.Page
			var err error
			if row.pagination == "apps-envelope" {
				one, err = esperruntime.UnwrapAppsEnvelope(first)
			} else {
				one, err = esperruntime.UnwrapLimitOffset(first)
			}
			if err != nil || one.Next == "" {
				t.Fatalf("first page = %#v, error = %v", one, err)
			}
			if row.pagination == "apps-envelope" {
				two, err = esperruntime.UnwrapAppsEnvelope(second)
			} else {
				two, err = esperruntime.UnwrapLimitOffset(second)
			}
			if err != nil || two.Next != "" {
				t.Fatalf("second page = %#v, error = %v", two, err)
			}
			merged, err := esperruntime.MarshalMergedResults(append(one.Results, two.Results...))
			var got, want any
			if err == nil {
				err = json.Unmarshal(merged, &got)
			}
			if err == nil {
				err = json.Unmarshal(readDeviceSupportFixture(t, row.name+"-success.golden"), &want)
			}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("merged output = %s, error = %v", merged, err)
			}
		})
	}
}

func TestDeviceSupportInputRules(t *testing.T) {
	for _, args := range [][]string{
		{"device-heartbeat-list", "list"},
		{"device-heartbeat-list", "list", "--last-seen-gt", "2026-08-24T00:00:00Z"},
		{"device-google-account-policy", "update", "device-1"},
		{"device-google-account-policy", "update", "device-1", "--body", "{"},
		{"google-account", "create", "--device", "device-1", "--tenant-id", "tenant-1"},
		{"google-account", "update", "device-1", "--google-device-id", "google-device-1"},
	} {
		assertDeviceSupportUsage(t, args)
	}

	root := NewRootCommand()
	create, _, err := root.Find([]string{"google-account", "create"})
	if err != nil || create.Flags().Lookup("device-id") != nil {
		t.Fatalf("google-account create must suppress body device-id flag")
	}
	policyCommand, _, err := root.Find([]string{"device-google-account-policy", "update"})
	if err != nil || policyCommand.Flags().Lookup("body") == nil || policyCommand.Flags().Lookup("applications") != nil {
		t.Fatalf("policy update must expose only explicit body input")
	}

	policy := `{"applications":[{"product_id":"app:com.example.fixture","app_action":"install"}]}`
	file := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(file, []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{policy, "@" + file, "-"} {
		input := ""
		if body == "-" {
			input = policy
		}
		assertDeviceSupportRequest(t, []string{"device-google-account-policy", "update", "device-1", "--body", body}, strings.NewReader(input), http.MethodPut, "/v2/devices/device-1/google-accounts/policy/", policy)
	}
	assertDeviceSupportRequest(t, []string{"google-account", "create", "--device", "device-1", "--tenant-id", "tenant-1", "--google-user-id", "google-user-1", "--account-type", "1"}, strings.NewReader(""), http.MethodPost, "/v2/devices/device-1/google-accounts/", `{"account_type":1,"device_id":"device-1","google_user_id":"google-user-1","tenant_id":"tenant-1"}`)
	assertDeviceSupportRequest(t, []string{"google-account", "create", "--device", "device-1"}, strings.NewReader(""), http.MethodPost, "/v2/devices/device-1/google-accounts/", "")
}

func TestDeviceSupportDestructiveCommandsDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	for _, row := range deviceSupportFixtures() {
		if !row.destructive {
			continue
		}
		command := configuredDeviceSupportCommand(t, server.URL)
		command.SetIn(strings.NewReader("no\n"))
		command.SetArgs(withoutYes(row.args))
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) = %v", row.args, err)
		}
	}
	if requests != 0 {
		t.Fatalf("declined requests = %d, want 0", requests)
	}
}

func executeDeviceSupportFixture(t *testing.T, row deviceSupportFixture, apiError bool) {
	t.Helper()
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != row.method || r.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, row.method, row.path)
		}
		if r.Header.Get("Authorization") != "Bearer fixture-key" || r.Header.Get("Accept") != "application/json" {
			t.Errorf("authorization = %q, accept = %q", r.Header.Get("Authorization"), r.Header.Get("Accept"))
		}
		if row.pagination != "" && requests == 2 {
			if got, want := r.URL.Query().Encode(), (url.Values{"offset": {"1"}}).Encode(); got != want {
				t.Errorf("second-page query = %q, want %q", got, want)
			}
		} else if got, want := r.URL.Query().Encode(), row.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != row.body {
			t.Errorf("body = %q, want %q", body, row.body)
		}
		if row.body != "" && r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q", r.Header.Get("Content-Type"))
		}
		name, status := row.name+"-success.json", row.status
		if apiError {
			name, status = row.name+"-api-error.json", row.errorStatus
		}
		if row.pagination != "" && requests == 2 {
			name = row.name + "-second-page.json"
		}
		response := readDeviceSupportFixture(t, name)
		if row.pagination != "" && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?offset=1"), 1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(response)
	}))
	defer server.Close()
	command := configuredDeviceSupportCommand(t, server.URL)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(row.args)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != row.errorStatus || esperruntime.ExitCode(err) != 1 || output.Len() != 0 {
			t.Fatalf("API error = %v, output = %q", err, output.String())
		}
		if want := readDeviceSupportFixture(t, row.name+"-api-error.json"); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %s, want %s", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if row.pagination != "" && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if want := readDeviceSupportFixture(t, row.name+"-success.golden"); !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.Bytes(), want)
	}
}

func assertDeviceSupportUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}

func assertDeviceSupportRequest(t *testing.T, args []string, input io.Reader, method, path, body string) {
	t.Helper()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != method || r.URL.Path != path {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		if string(data) != body {
			t.Errorf("body = %q, want %q", data, body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(readDeviceSupportFixture(t, "v2-google-account-create-success.json"))
	}))
	defer server.Close()
	command := configuredDeviceSupportCommand(t, server.URL)
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

func configuredDeviceSupportCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func deviceSupportValidateFixture(t *testing.T, document, schema map[string]any, name string) {
	t.Helper()
	var value any
	if err := json.Unmarshal(readDeviceSupportFixture(t, name), &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

func readDeviceSupportFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "device-support", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
