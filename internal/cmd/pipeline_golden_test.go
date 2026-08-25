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

type pipelineFixtureTest struct {
	key, name, method, path, body, fixture string
	arguments                              []string
	query                                  url.Values
	status, errorStatus                    int
	all, destructive                       bool
}

func pipelineFixtureTests() []pipelineFixtureTest {
	return []pipelineFixtureTest{
		{"pipelines-v0 GET /pipelines/v0/pipelines/", "pipeline list", http.MethodGet, "/pipelines/v0/pipelines/", "", "pipeline-list", []string{"pipeline", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"pipelines-v0 POST /pipelines/v0/pipelines/", "pipeline create", http.MethodPost, "/pipelines/v0/pipelines/", `{"name":"Pipeline fixture"}`, "pipeline-create", []string{"pipeline", "create", "--name", "Pipeline fixture", "--json"}, nil, 200, 400, false, false},
		{"pipelines-v0 GET /pipelines/v0/pipelines/{pipeline_id}/", "pipeline get", http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/", "", "pipeline-get", []string{"pipeline", "get", "pipeline-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 PUT /pipelines/v0/pipelines/{pipeline_id}/", "pipeline update", http.MethodPut, "/pipelines/v0/pipelines/pipeline-1/", `{"name":"Pipeline updated"}`, "pipeline-update", []string{"pipeline", "update", "pipeline-1", "--name", "Pipeline updated", "--json"}, nil, 200, 400, false, false},
		{"pipelines-v0 DELETE /pipelines/v0/pipelines/{pipeline_id}/", "pipeline delete", http.MethodDelete, "/pipelines/v0/pipelines/pipeline-1/", "", "pipeline-delete", []string{"pipeline", "delete", "pipeline-1", "--yes", "--json"}, nil, 200, 401, false, true},
		{"pipelines-v0 GET /pipelines/v0/pipelines/{pipeline_id}/stages/", "stage list", http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/stages/", "", "stage-list", []string{"stage", "list", "--pipeline", "pipeline-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"pipelines-v0 POST /pipelines/v0/pipelines/{pipeline_id}/stages/", "stage create", http.MethodPost, "/pipelines/v0/pipelines/pipeline-1/stages/", `{"name":"Stage fixture"}`, "stage-create", []string{"stage", "create", "--pipeline", "pipeline-1", "--name", "Stage fixture", "--json"}, nil, 200, 400, false, false},
		{"pipelines-v0 GET /pipelines/v0/pipelines/{pipeline_id}/stages/{stage_id}/", "stage get", http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/stages/stage-1/", "", "stage-get", []string{"stage", "get", "stage-1", "--pipeline", "pipeline-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 PUT /pipelines/v0/pipelines/{pipeline_id}/stages/{stage_id}/", "stage update", http.MethodPut, "/pipelines/v0/pipelines/pipeline-1/stages/stage-1/", `{"name":"Stage updated"}`, "stage-update", []string{"stage", "update", "stage-1", "--pipeline", "pipeline-1", "--name", "Stage updated", "--json"}, nil, 200, 400, false, false},
		{"pipelines-v0 DELETE /pipelines/v0/pipelines/{pipeline_id}/stages/{stage_id}/", "stage delete", http.MethodDelete, "/pipelines/v0/pipelines/pipeline-1/stages/stage-1/", "", "stage-delete", []string{"stage", "delete", "stage-1", "--pipeline", "pipeline-1", "--yes", "--json"}, nil, 200, 401, false, true},
		{"pipelines-v0 GET /pipelines/v0/stages/{stage_id}/targetlists/", "target-list list by stage", http.MethodGet, "/pipelines/v0/stages/stage-1/targetlists/", "", "target-list-list-stage", []string{"target-list", "list", "--stage", "stage-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"pipelines-v0 POST /pipelines/v0/stages/{stage_id}/targetlists/", "target-list add by stage", http.MethodPost, "/pipelines/v0/stages/stage-1/targetlists/", `{"target_list_id":"target-list-1"}`, "target-list-add-stage", []string{"target-list", "add", "--stage", "stage-1", "--target-list-id", "target-list-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 DELETE /pipelines/v0/stages/{stage_id}/targetlists/{targetlist_id}/", "target-list delete by stage", http.MethodDelete, "/pipelines/v0/stages/stage-1/targetlists/target-list-1/", "", "target-list-delete-stage", []string{"target-list", "delete", "target-list-1", "--stage", "stage-1", "--yes", "--json"}, nil, 200, 401, false, true},
		{"pipelines-v0 GET /pipelines/v0/pipelines/{pipeline_id}/targetlists/", "target-list list by pipeline", http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/targetlists/", "", "target-list-list-pipeline", []string{"target-list", "list", "--pipeline", "pipeline-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"pipelines-v0 POST /pipelines/v0/pipelines/{pipeline_id}/targetlists/", "target-list add by pipeline", http.MethodPost, "/pipelines/v0/pipelines/pipeline-1/targetlists/", `{"name":"Target list fixture","target_list_type":"DEVICES"}`, "target-list-add-pipeline", []string{"target-list", "add", "--pipeline", "pipeline-1", "--name", "Target list fixture", "--target-list-type", "DEVICES", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 GET /pipelines/v0/pipelines/{pipeline_id}/targetlists/{targetlist_id}/", "target-list get", http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/", "", "target-list-get", []string{"target-list", "get", "target-list-1", "--pipeline", "pipeline-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 PUT /pipelines/v0/pipelines/{pipeline_id}/targetlists/{targetlist_id}/", "target-list update", http.MethodPut, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/", `{"name":"Target list updated"}`, "target-list-update", []string{"target-list", "update", "target-list-1", "--pipeline", "pipeline-1", "--name", "Target list updated", "--json"}, nil, 200, 400, false, false},
		{"pipelines-v0 DELETE /pipelines/v0/pipelines/{pipeline_id}/targetlists/{targetlist_id}/", "target-list delete by pipeline", http.MethodDelete, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/", "", "target-list-delete-pipeline", []string{"target-list", "delete", "target-list-1", "--pipeline", "pipeline-1", "--yes", "--json"}, nil, 200, 401, false, true},
		{"pipelines-v0 GET /pipelines/v0/pipelines/{pipeline_id}/targetlists/{targetlist_id}/targets/", "target list", http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/targets/", "", "target-list", []string{"target", "list", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"pipelines-v0 POST /pipelines/v0/pipelines/{pipeline_id}/targetlists/{targetlist_id}/targets/", "target create", http.MethodPost, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/targets/", `{"device_id":"device-1","device_name":"Device fixture"}`, "target-create", []string{"target", "create", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--device-id", "device-1", "--device-name", "Device fixture", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 GET /pipelines/v0/pipelines/{pipeline_id}/targetlists/{targetlist_id}/targets/{target_id}/", "target get", http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/targets/target-1/", "", "target-get", []string{"target", "get", "target-1", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 PUT /pipelines/v0/pipelines/{pipeline_id}/targetlists/{targetlist_id}/targets/{target_id}/", "target update", http.MethodPut, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/targets/target-1/", `{"device_name":"Device updated"}`, "target-update", []string{"target", "update", "target-1", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--device-name", "Device updated", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 DELETE /pipelines/v0/pipelines/{pipeline_id}/targetlists/{targetlist_id}/targets/{target_id}/", "target delete", http.MethodDelete, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/targets/target-1/", "", "target-delete", []string{"target", "delete", "target-1", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--yes", "--json"}, nil, 200, 401, false, true},
		{"pipelines-v0 POST /pipelines/v0/pipelines/{pipeline_id}/targetlists/{targetlist_id}/targets-bulk/", "target-bulk add", http.MethodPost, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/targets-bulk/", `["device-1"]`, "target-bulk-add", []string{"target-bulk", "bulk-add", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--body", `["device-1"]`, "--json"}, nil, 200, 500, false, false},
		{"pipelines-v0 GET /pipelines/v0/runs/", "pipeline-run list", http.MethodGet, "/pipelines/v0/runs/", "", "pipeline-run-list", []string{"pipeline-run", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"pipelines-v0 GET /pipelines/v0/pipelines/{pipeline_id}/runs/", "pipeline-run list by pipeline", http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/runs/", "", "pipeline-run-list-pipeline", []string{"pipeline-run", "list", "--pipeline", "pipeline-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"pipelines-v0 POST /pipelines/v0/pipelines/{pipeline_id}/runs/", "pipeline-run create", http.MethodPost, "/pipelines/v0/pipelines/pipeline-1/runs/", `{}`, "pipeline-run-create", []string{"pipeline-run", "create", "--pipeline", "pipeline-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 GET /pipelines/v0/pipelines/{pipeline_id}/runs/{pipeline_run_id}/", "pipeline-run get", http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/runs/pipeline-run-1/", "", "pipeline-run-get", []string{"pipeline-run", "get", "pipeline-run-1", "--pipeline", "pipeline-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 PUT /pipelines/v0/pipelines/{pipeline_id}/runs/{pipeline_run_id}/", "pipeline-run update", http.MethodPut, "/pipelines/v0/pipelines/pipeline-1/runs/pipeline-run-1/", `{"status":"success"}`, "pipeline-run-update", []string{"pipeline-run", "update", "pipeline-run-1", "--pipeline", "pipeline-1", "--status", "success", "--json"}, nil, 200, 400, false, false},
		{"pipelines-v0 DELETE /pipelines/v0/pipelines/{pipeline_id}/runs/{pipeline_run_id}/", "pipeline-run delete", http.MethodDelete, "/pipelines/v0/pipelines/pipeline-1/runs/pipeline-run-1/", "", "pipeline-run-delete", []string{"pipeline-run", "delete", "pipeline-run-1", "--pipeline", "pipeline-1", "--yes", "--json"}, nil, 200, 400, false, true},
		{"pipelines-v0 GET /pipelines/v0/runs/{pipeline_run_id}/stageruns/", "stage-run list", http.MethodGet, "/pipelines/v0/runs/pipeline-run-1/stageruns/", "", "stage-run-list", []string{"stage-run", "list", "--pipeline-run", "pipeline-run-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"pipelines-v0 GET /pipelines/v0/runs/{pipeline_run_id}/stageruns/{stage_run_id}/", "stage-run get", http.MethodGet, "/pipelines/v0/runs/pipeline-run-1/stageruns/stage-run-1/", "", "stage-run-get", []string{"stage-run", "get", "stage-run-1", "--pipeline-run", "pipeline-run-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 PUT /pipelines/v0/runs/{pipeline_run_id}/stageruns/{stage_run_id}/", "stage-run update", http.MethodPut, "/pipelines/v0/runs/pipeline-run-1/stageruns/stage-run-1/", `{"status":"success"}`, "stage-run-update", []string{"stage-run", "update", "stage-run-1", "--pipeline-run", "pipeline-run-1", "--status", "success", "--json"}, nil, 200, 400, false, false},
		{"pipelines-v0 GET /pipelines/v0/stageruns/{stage_run_id}/targetruns/", "target-run list", http.MethodGet, "/pipelines/v0/stageruns/stage-run-1/targetruns/", "", "target-run-list", []string{"target-run", "list", "--stage-run", "stage-run-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"pipelines-v0 POST /pipelines/v0/stageruns/{stage_run_id}/targetruns/", "target-run create", http.MethodPost, "/pipelines/v0/stageruns/stage-run-1/targetruns/", `{"target_id":"target-1"}`, "target-run-create", []string{"target-run", "create", "--stage-run", "stage-run-1", "--target-id", "target-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 GET /pipelines/v0/stageruns/{stage_run_id}/targetruns/{target_run_id}/", "target-run get", http.MethodGet, "/pipelines/v0/stageruns/stage-run-1/targetruns/target-run-1/", "", "target-run-get", []string{"target-run", "get", "target-run-1", "--stage-run", "stage-run-1", "--json"}, nil, 200, 401, false, false},
		{"pipelines-v0 PUT /pipelines/v0/stageruns/{stage_run_id}/targetruns/{target_run_id}/", "target-run update", http.MethodPut, "/pipelines/v0/stageruns/stage-run-1/targetruns/target-run-1/", `{"status":"success"}`, "target-run-update", []string{"target-run", "update", "target-run-1", "--stage-run", "stage-run-1", "--status", "success", "--json"}, nil, 200, 400, false, false},
	}
}

func TestPipelineOperationCoverage(t *testing.T) {
	nouns := map[string]bool{"pipeline": true, "pipeline-run": true, "stage": true, "stage-run": true, "target": true, "target-bulk": true, "target-list": true, "target-run": true}
	expected := map[string]bool{}
	for _, test := range pipelineFixtureTests() {
		if expected[test.key] {
			t.Fatalf("duplicate explicit fixture row %s", test.key)
		}
		expected[test.key] = true
	}
	if len(expected) != 37 {
		t.Fatalf("fixture rows = %d, want 37", len(expected))
	}
	actual := map[string]bool{}
	for _, operation := range generated.Operations() {
		if operation.Generation == "pipelines-v0" && nouns[operation.Noun] {
			actual[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if len(actual) != 37 || !reflect.DeepEqual(expected, actual) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(expected), len(actual))
	}
}

func TestPipelineCommandsGoldenFixtures(t *testing.T) {
	for _, test := range pipelineFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executePipelineFixture(t, test, false) })
	}
}

func TestPipelineCommandsAPIErrors(t *testing.T) {
	for _, test := range pipelineFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executePipelineFixture(t, test, true) })
	}
}

func TestPipelineInputValidation(t *testing.T) {
	for _, arguments := range [][]string{
		{"pipeline", "update", "pipeline-1"},
		{"stage", "update", "stage-1", "--pipeline", "pipeline-1"},
		{"target-list", "update", "target-list-1", "--pipeline", "pipeline-1"},
		{"target", "update", "target-1", "--pipeline", "pipeline-1", "--target-list", "target-list-1"},
		{"target-bulk", "bulk-add", "--pipeline", "pipeline-1", "--target-list", "target-list-1"},
		{"pipeline-run", "update", "pipeline-run-1", "--pipeline", "pipeline-1"},
		{"stage-run", "update", "stage-run-1", "--pipeline-run", "pipeline-run-1"},
		{"target-run", "update", "target-run-1", "--stage-run", "stage-run-1"},
		{"target-list", "list"},
		{"target-list", "list", "--stage", "stage-1", "--pipeline", "pipeline-1"},
		{"pipeline", "update", "pipeline-1", "--body", `{}`, "--name", "conflict"},
	} {
		command := NewRootCommand()
		command.SetArgs(arguments)
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) error = %v", arguments, err)
		}
	}
}

func TestPipelineDestructiveCommandsDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	for _, test := range pipelineFixtureTests() {
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

func executePipelineFixture(t *testing.T, test pipelineFixtureTest, apiError bool) {
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
		assertPipelineBody(t, request, test.body)
		response := readPipelineFixture(t, fixture)
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
		if !errors.As(err, &value) || value.StatusCode != status || esperruntime.ExitCode(err) != 1 {
			t.Fatalf("API error = %v", err)
		}
		if want := readPipelineFixture(t, fixture); !bytes.Equal(value.Body, want) {
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
	if err := json.Unmarshal(readPipelineFixture(t, test.fixture+"-success.golden"), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", test.fixture, output.Bytes())
	}
}

func assertPipelineBody(t *testing.T, request *http.Request, want string) {
	t.Helper()
	data, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if want == "" {
		if len(data) != 0 {
			t.Errorf("body = %q, want empty", data)
		}
		return
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(data, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("body = %s, want %s", data, want)
	}
}

func readPipelineFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "pipeline", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
