package cmd

import (
	"bytes"
	"context"
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

type commandOperationFixture struct {
	key, name, method, path, body string
	args                          []string
	query                         url.Values
	status, errorStatus           int
	pagination                    string
	destructive                   bool
}

func commandOperationFixtures() []commandOperationFixture {
	return []commandOperationFixture{
		{"v0 GET /v0/enterprise/{enterprise_id}/command/", "command-list", "GET", "/v0/enterprise/tenant-1/command/", "", []string{"api", "v0", "command", "list", "--enterprise", "tenant-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 400, "limit-offset", false},
		{"v0 POST /v0/enterprise/{enterprise_id}/command/", "command-create", "POST", "/v0/enterprise/tenant-1/command/", `{"command":"REBOOT"}`, []string{"command", "create", "--enterprise", "tenant-1", "--command", "REBOOT", "--json"}, nil, 201, 400, "", false},
		{"v0 GET /v0/enterprise/{enterprise_id}/command/{request_id}/status/", "status-get", "GET", "/v0/enterprise/tenant-1/command/request-1/status/", "", []string{"status", "get", "tenant-1", "request-1", "--all", "--json"}, nil, 200, 400, "limit-offset", false},
		{"v0 GET /v0/enterprise/{enterprise_id}/device/{device_id}/command-history/", "command-history-get", "GET", "/v0/enterprise/tenant-1/device/device-1/command-history/", "", []string{"command-history", "get", "tenant-1", "device-1", "--all", "--json"}, nil, 200, 400, "limit-offset", false},
		{"v0 GET /v0/operations/", "operation-list-list", "GET", "/v0/operations/", "", []string{"operation-list", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 400, "apps-envelope", false},
		{"v0 POST /v0/operations/", "operation-create", "POST", "/v0/operations/", `{"operation_type":"REBOOT"}`, []string{"operation", "create", "--operation-type", "REBOOT", "--json"}, nil, 201, 400, "", false},
		{"v0 GET /v0/operations/{operationId}/", "operation-get", "GET", "/v0/operations/operation-1/", "", []string{"operation", "get", "operation-1", "--json"}, nil, 200, 400, "", false},
		{"v0 PUT /v0/operations/{operationId}/", "operation-update", "PUT", "/v0/operations/operation-1/", `{"reason":"fixture","state":"CANCELLED"}`, []string{"operation", "update", "operation-1", "--state", "CANCELLED", "--reason", "fixture", "--yes", "--json"}, nil, 200, 400, "", true},
		{"v0 GET /v0/devices/{deviceId}/operations/", "operation-list-device", "GET", "/v0/devices/device-1/operations/", "", []string{"operation", "list", "--device", "device-1", "--all", "--json"}, nil, 200, 400, "limit-offset", false},
		{"v0 GET /v0/operations/{operationsId}/devices/{deviceId}/", "device-operation-get", "GET", "/v0/operations/operation-1/devices/device-1/", "", []string{"device-operation", "get", "operation-1", "device-1", "--json"}, nil, 200, 400, "", false},
		{"v0 PUT /v0/operations/{operationsId}/devices/{deviceId}/", "device-operation-update", "PUT", "/v0/operations/operation-1/devices/device-1/", `{"state":"CANCELLATION_REQUESTED"}`, []string{"device-operation", "update", "operation-1", "device-1", "--state", "CANCELLATION_REQUESTED", "--yes", "--json"}, nil, 200, 400, "", true},
		{"v0 GET /commands/v0/commands/", "command-request-list", "GET", "/commands/v0/commands/", "", []string{"command-request", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 400, "apps-envelope", false},
		{"v0 POST /commands/v0/commands/", "command-request-create", "POST", "/commands/v0/commands/", `{"command":"REBOOT"}`, []string{"command-request", "create", "--command", "REBOOT", "--json"}, nil, 201, 400, "", false},
		{"v0 GET /commands/v0/commands/{id}/", "command-request-get", "GET", "/commands/v0/commands/request-1/", "", []string{"command-request", "get", "request-1", "--json"}, nil, 200, 400, "", false},
		{"v0 GET /commands/v0/commands/{id}/stats/", "stat-list", "GET", "/commands/v0/commands/request-1/stats/", "", []string{"stat", "list", "--command", "request-1", "--json"}, nil, 200, 400, "", false},
		{"v0 GET /commands/v0/status/", "command-request-status-list", "GET", "/commands/v0/status/", "", []string{"command-request-status", "list", "--device", "device-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"device": {"device-1"}, "limit": {"1"}, "offset": {"0"}}, 200, 400, "apps-envelope", false},
		{"v0 GET /commands/v0/status/{id}/", "command-status-get", "GET", "/commands/v0/status/status-1/", "", []string{"command-status", "get", "status-1", "--json"}, nil, 200, 400, "", false},
		{"v0 PUT /commands/v0/status/{id}/", "command-status-update", "PUT", "/commands/v0/status/status-1/", `{"state":"Command Cancelled"}`, []string{"command-status", "update", "status-1", "--state", "Command Cancelled", "--yes", "--json"}, nil, 200, 400, "", true},
		{"v2 GET /v2/command-inbox/", "command-inbox-get", "GET", "/v2/command-inbox/", "", []string{"command-inbox", "get", "--device-id", "device-1", "--json"}, url.Values{"device_id": {"device-1"}}, 200, 400, "", false},
		{"v2 GET /v2/converge/{id}/commands", "converge-command-list", "GET", "/v2/converge/converge-1/commands", "", []string{"command", "list", "--converge", "converge-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 400, "apps-envelope", false},
		{"legacy GET /enterprise/{enterprise_id}/device/{device_id}/status/", "legacy-status-get", "GET", "/enterprise/tenant-1/device/device-1/status/", "", []string{"api", "legacy", "status", "get", "tenant-1", "device-1", "--latest-event", "true", "--all", "--json"}, url.Values{"latest_event": {"true"}}, 200, 400, "limit-offset", false},
		{"legacy GET /enterprise/{enterprise_id}/devicegroup/{group_id}/command/", "legacy-command-list", "GET", "/enterprise/tenant-1/devicegroup/group-1/command/", "", []string{"api", "legacy", "command", "list", "--enterprise", "tenant-1", "--device-group", "group-1", "--all", "--json"}, nil, 200, 400, "limit-offset", false},
		{"legacy POST /enterprise/{enterprise_id}/devicegroup/{group_id}/command/", "legacy-command-create", "POST", "/enterprise/tenant-1/devicegroup/group-1/command/", `{"command":"REBOOT"}`, []string{"api", "legacy", "command", "create", "--enterprise", "tenant-1", "--device-group", "group-1", "--command", "REBOOT", "--json"}, nil, 201, 400, "", false},
		{"pipelines-v0 GET /pipelines/v0/stages/{stage_id}/operationlists/", "pipeline-operation-list-list", "GET", "/pipelines/v0/stages/stage-1/operationlists/", "", []string{"operation-list", "list", "--stage", "stage-1", "--all", "--json"}, nil, 200, 400, "apps-envelope", false},
		{"pipelines-v0 POST /pipelines/v0/stages/{stage_id}/operationlists/", "pipeline-operation-list-add", "POST", "/pipelines/v0/stages/stage-1/operationlists/", `{"name":"fixture"}`, []string{"operation-list", "add", "--stage", "stage-1", "--name", "fixture", "--json"}, nil, 200, 400, "", false},
		{"pipelines-v0 PUT /pipelines/v0/stages/{stage_id}/operationlists/{operationlist_id}/", "pipeline-operation-list-update", "PUT", "/pipelines/v0/stages/stage-1/operationlists/list-1/", `{"name":"fixture"}`, []string{"operation-list", "update", "list-1", "--stage", "stage-1", "--name", "fixture", "--json"}, nil, 200, 400, "", false},
		{"pipelines-v0 DELETE /pipelines/v0/stages/{stage_id}/operationlists/{operationlist_id}/", "pipeline-operation-list-delete", "DELETE", "/pipelines/v0/stages/stage-1/operationlists/list-1/", "", []string{"operation-list", "delete", "list-1", "--stage", "stage-1", "--yes", "--json"}, nil, 200, 400, "", true},
		{"pipelines-v0 GET /pipelines/v0/operationlists/{operationlist_id}/operations/", "pipeline-operation-list-scope", "GET", "/pipelines/v0/operationlists/list-1/operations/", "", []string{"pipeline-operation", "list", "--operation-list", "list-1", "--all", "--json"}, nil, 200, 400, "apps-envelope", false},
		{"pipelines-v0 POST /pipelines/v0/operationlists/{operationlist_id}/operations/", "pipeline-operation-add", "POST", "/pipelines/v0/operationlists/list-1/operations/", `{"operation_type":"REBOOT"}`, []string{"pipeline-operation", "add", "--operation-list", "list-1", "--operation-type", "REBOOT", "--json"}, nil, 200, 400, "", false},
		{"pipelines-v0 GET /pipelines/v0/operationlists/{operationlist_id}/operations/{operation_id}/", "pipeline-operation-get", "GET", "/pipelines/v0/operationlists/list-1/operations/operation-1/", "", []string{"pipeline-operation", "get", "operation-1", "--operation-list", "list-1", "--json"}, nil, 200, 400, "", false},
		{"pipelines-v0 PUT /pipelines/v0/operationlists/{operationlist_id}/operations/{operation_id}/", "pipeline-operation-update", "PUT", "/pipelines/v0/operationlists/list-1/operations/operation-1/", `{"operation_type":"REBOOT"}`, []string{"pipeline-operation", "update", "operation-1", "--operation-list", "list-1", "--operation-type", "REBOOT", "--json"}, nil, 200, 400, "", false},
		{"pipelines-v0 DELETE /pipelines/v0/operationlists/{operationlist_id}/operations/{operation_id}/", "pipeline-operation-delete", "DELETE", "/pipelines/v0/operationlists/list-1/operations/operation-1/", "", []string{"pipeline-operation", "delete", "operation-1", "--operation-list", "list-1", "--yes", "--json"}, nil, 200, 400, "", true},
		{"pipelines-v0 GET /pipelines/v0/stageruns/{stage_run_id}/operations/", "pipeline-operation-stage-run-list", "GET", "/pipelines/v0/stageruns/stage-run-1/operations/", "", []string{"pipeline-operation", "list", "--stage-run", "stage-run-1", "--all", "--json"}, nil, 200, 400, "apps-envelope", false},
		{"pipelines-v0 GET /pipelines/v0/stageruns/{stage_run_id}/operations/{stage_run_operation_id}/", "pipeline-operation-stage-run-get", "GET", "/pipelines/v0/stageruns/stage-run-1/operations/operation-1/", "", []string{"pipeline-operation", "get", "operation-1", "--stage-run", "stage-run-1", "--json"}, nil, 200, 400, "", false},
		{"pipelines-v0 GET /pipelines/v0/stageruns/{stage_run_id}/targetruns/{target_run_id}/command/", "pipeline-command-list", "GET", "/pipelines/v0/stageruns/stage-run-1/targetruns/target-run-1/command/", "", []string{"pipeline-command", "list", "--stage-run", "stage-run-1", "--target-run", "target-run-1", "--json"}, nil, 200, 400, "", false},
		{"pipelines-v0 POST /pipelines/v0/stageruns/{stage_run_id}/targetruns/{target_run_id}/command/", "pipeline-command-create", "POST", "/pipelines/v0/stageruns/stage-run-1/targetruns/target-run-1/command/", `{}`, []string{"pipeline-command", "create", "--stage-run", "stage-run-1", "--target-run", "target-run-1", "--json"}, nil, 200, 400, "", false},
		{"pipelines-v0 PUT /pipelines/v0/stageruns/{stage_run_id}/targetruns/{target_run_id}/command/{command_id}/", "pipeline-command-update", "PUT", "/pipelines/v0/stageruns/stage-run-1/targetruns/target-run-1/command/command-1/", `{"request_status":"cancelled"}`, []string{"pipeline-command", "update", "command-1", "--stage-run", "stage-run-1", "--target-run", "target-run-1", "--request-status", "cancelled", "--yes", "--json"}, nil, 200, 400, "", true},
	}
}

func TestCommandOperationCoverage(t *testing.T) {
	want := map[string]bool{}
	for _, row := range commandOperationFixtures() {
		if want[row.key] {
			t.Fatalf("duplicate fixture row %s", row.key)
		}
		want[row.key] = true
	}
	if len(want) != 37 {
		t.Fatalf("fixture rows = %d, want 37", len(want))
	}
	nouns := map[string]bool{
		"command": true, "command-history": true, "command-inbox": true,
		"command-request": true, "command-request-status": true, "command-status": true,
		"device-operation": true, "operation": true, "operation-list": true,
		"pipeline-command": true, "pipeline-operation": true, "stat": true, "status": true,
	}
	got := map[string]bool{}
	for _, operation := range generated.Operations() {
		if nouns[operation.Noun] {
			got[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if len(got) != 37 || !reflect.DeepEqual(want, got) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(want), len(got))
	}
}

func TestCommandOperationFixturesMatchResponseContracts(t *testing.T) {
	documents := map[string]map[string]any{}
	for _, file := range []string{"v0.yaml", "v2.yaml", "legacy.yaml", "pipelines-v0.yaml"} {
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
	for _, row := range commandOperationFixtures() {
		generation := strings.SplitN(row.key, " ", 2)[0]
		document := documents[generation]
		parts := strings.SplitN(row.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		if row.status != http.StatusNotModified {
			fixtureSchemaValidateCommandOperationFile(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		}
		if row.pagination != "" {
			fixtureSchemaValidateCommandOperationFile(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-second-page.json")
		}
		errorSchema := fixtureSchemaResponseForStatus(document, operation, row.errorStatus)
		if errorSchema != nil {
			fixtureSchemaValidateCommandOperationFile(t, document, errorSchema, row.name+"-api-error.json")
			continue
		}
		var value any
		if err := json.Unmarshal(readCommandOperationFixture(t, row.name+"-api-error.json"), &value); err != nil {
			t.Fatalf("%s-api-error.json is not JSON: %v", row.name, err)
		}
	}
}

func fixtureSchemaValidateCommandOperationFile(t *testing.T, document, schema map[string]any, name string) {
	t.Helper()
	data := readCommandOperationFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

func TestCommandOperationGoldenFixtures(t *testing.T) {
	for _, row := range commandOperationFixtures() {
		t.Run(row.name, func(t *testing.T) { executeCommandOperationFixture(t, row, false) })
	}
}

func TestCommandOperationPaginationFixtures(t *testing.T) {
	flows := []struct{ name, kind string }{
		{"command-list", "limit-offset"}, {"status-get", "limit-offset"}, {"command-history-get", "limit-offset"},
		{"operation-list-list", "apps-envelope"}, {"operation-list-device", "limit-offset"}, {"command-request-list", "apps-envelope"},
		{"command-request-status-list", "apps-envelope"}, {"converge-command-list", "apps-envelope"}, {"legacy-status-get", "limit-offset"},
		{"legacy-command-list", "limit-offset"}, {"pipeline-operation-list-list", "apps-envelope"},
		{"pipeline-operation-list-scope", "apps-envelope"}, {"pipeline-operation-stage-run-list", "apps-envelope"},
	}
	if len(flows) != 13 {
		t.Fatalf("pagination flows = %d, want 13", len(flows))
	}
	for _, flow := range flows {
		t.Run(flow.name, func(t *testing.T) {
			first := readCommandOperationFixture(t, flow.name+"-success.json")
			second := readCommandOperationFixture(t, flow.name+"-second-page.json")
			var one, two esperruntime.Page
			var err error
			if flow.kind == "apps-envelope" {
				one, err = esperruntime.UnwrapAppsEnvelope(first)
			} else {
				one, err = esperruntime.UnwrapLimitOffset(first)
			}
			if err != nil || one.Next == "" {
				t.Fatalf("first page = %#v, error = %v", one, err)
			}
			if flow.kind == "apps-envelope" {
				two, err = esperruntime.UnwrapAppsEnvelope(second)
			} else {
				two, err = esperruntime.UnwrapLimitOffset(second)
			}
			if err != nil || two.Next != "" {
				t.Fatalf("second page = %#v, error = %v", two, err)
			}
			merged, err := esperruntime.MarshalMergedResults(append(one.Results, two.Results...))
			if err != nil {
				t.Fatal(err)
			}
			if len(merged) == 0 {
				t.Fatal("merged output is empty")
			}
		})
	}
}
func TestCommandOperationAPIErrors(t *testing.T) {
	for _, row := range commandOperationFixtures() {
		t.Run(row.name, func(t *testing.T) { executeCommandOperationFixture(t, row, true) })
	}
}

func TestCommandOperationInputValidation(t *testing.T) {
	for _, args := range [][]string{
		{"operation", "create"}, {"command-request", "create"}, {"operation", "update", "operation-1", "--state", "CANCELLED"},
		{"pipeline-operation", "list"}, {"pipeline-operation", "list", "--operation-list", "list-1", "--stage-run", "stage-run-1"},
		{"pipeline-operation", "update", "operation-1", "--operation-list", "list-1"}, {"operation-list", "update", "list-1", "--stage", "stage-1"},
		{"command", "create", "--enterprise", "tenant-1", "--body", `{"command":"REBOOT"}`, "--command", "REBOOT"},
		{"operation", "create", "--body", "{"},
		{"command-request-status", "list"},
	} {
		assertCommandOperationUsage(t, args)
	}
}

func TestCommandOperationBodyFileAndStdin(t *testing.T) {
	file := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(file, []byte(`{"operation_type":"REBOOT"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"@" + file, "-"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, _ := io.ReadAll(r.Body)
			if string(data) != `{"operation_type":"REBOOT"}` {
				t.Errorf("body = %s", data)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		}))
		command := configuredCommandOperationCommand(t, server.URL)
		command.SetIn(strings.NewReader(`{"operation_type":"REBOOT"}`))
		command.SetArgs([]string{"operation", "create", "--body", body, "--json"})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		server.Close()
	}
}

func TestCommandOperationDestructiveCommandsDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	for _, row := range commandOperationFixtures() {
		if !row.destructive {
			continue
		}
		command := configuredCommandOperationCommand(t, server.URL)
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

func TestCommandOperationCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command := configuredCommandOperationCommand(t, "http://127.0.0.1:1")
	command.SetContext(ctx)
	command.SetArgs([]string{"operation", "get", "operation-1", "--json"})
	if err := command.Execute(); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled command error = %v", err)
	}
}

func TestCommandOperationBodyless304(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotModified) }))
	defer server.Close()
	command := configuredCommandOperationCommand(t, server.URL)
	command.SetArgs([]string{"command-status", "update", "status-1", "--state", "Command Cancelled", "--yes", "--json"})
	err := command.Execute()
	var value *esperruntime.APIError
	if !errors.As(err, &value) || value.StatusCode != http.StatusNotModified || len(value.Body) != 0 {
		t.Fatalf("bodyless 304 error = %#v", err)
	}
}

func executeCommandOperationFixture(t *testing.T, row commandOperationFixture, apiError bool) {
	t.Helper()
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != row.method || r.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, row.method, row.path)
		}
		if r.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("accept = %q", r.Header.Get("Accept"))
		}
		if row.pagination != "" && requests == 2 {
			if got, want := r.URL.Query().Encode(), (url.Values{"offset": {"1"}}).Encode(); got != want {
				t.Errorf("second page query = %q, want %q", got, want)
			}
		} else if got, want := r.URL.Query().Encode(), row.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		data, _ := io.ReadAll(r.Body)
		if string(data) != row.body {
			t.Errorf("body = %q, want %q", data, row.body)
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
		response := readCommandOperationFixture(t, name)
		if row.pagination != "" && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?offset=1"), 1)
		}
		if status != http.StatusNotModified {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		_, _ = w.Write(response)
	}))
	defer server.Close()
	command := configuredCommandOperationCommand(t, server.URL)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(row.args)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != row.errorStatus || esperruntime.ExitCode(err) != 1 {
			t.Fatalf("API error = %v", err)
		}
		if !bytes.Equal(value.Body, readCommandOperationFixture(t, row.name+"-api-error.json")) {
			t.Fatalf("API error body = %s", value.Body)
		}
		if output.Len() != 0 {
			t.Fatalf("API error output = %q, want empty", output.Bytes())
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if row.pagination != "" && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if want := readCommandOperationFixture(t, row.name+"-success.golden"); !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.Bytes(), want)
	}
}

func assertCommandOperationUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}
func configuredCommandOperationCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}
func readCommandOperationFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "command-operation", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
