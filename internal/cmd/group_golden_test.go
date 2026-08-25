package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestGroupCommandsGoldenFixtures(t *testing.T) {
	image := filepath.Join(t.TempDir(), "thumbnail.png")
	if err := os.WriteFile(image, []byte("group thumbnail fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []groupFixtureTest{
		{"device group list", []string{"device-group", "list", "--enterprise", "enterprise-1", "--limit", "1", "--offset", "0", "--all", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "device-group-list-success.json", "device-group-list-success.golden", http.StatusOK, true, false},
		{"device group create", []string{"device-group", "create", "--enterprise", "enterprise-1", "--name", "Fixture Group", "--json"}, http.MethodPost, "/enterprise/enterprise-1/devicegroup/", nil, `{"name":"Fixture Group"}`, "device-group-create-success.json", "device-group-create-success.golden", http.StatusCreated, false, false},
		{"group get", []string{"group", "get", "group-1", "enterprise-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/group-1/", nil, "", "group-get-success.json", "group-get-success.golden", http.StatusOK, false, false},
		{"group update", []string{"group", "update", "group-1", "enterprise-1", "--action", "rename", "--name", "Renamed Group", "--json"}, http.MethodPut, "/enterprise/enterprise-1/devicegroup/group-1/", url.Values{"action": {"rename"}}, `{"name":"Renamed Group"}`, "group-update-success.json", "group-update-success.golden", http.StatusOK, false, false},
		{"group delete", []string{"group", "delete", "group-1", "enterprise-1", "--yes", "--json"}, http.MethodDelete, "/enterprise/enterprise-1/devicegroup/group-1/", nil, "", "group-delete-success.json", "group-delete-success.golden", http.StatusNoContent, false, false},
		{"group partial update", []string{"group", "partial-update", "group-1", "enterprise-1", "--name", "Partial Group", "--json"}, http.MethodPatch, "/enterprise/enterprise-1/devicegroup/group-1/", nil, `{"name":"Partial Group"}`, "group-partial-update-success.json", "group-partial-update-success.golden", http.StatusOK, false, false},
		{"group eventfeed list", []string{"group-eventfeed", "list", "--enterprise", "enterprise-1", "--group", "group-1", "--limit", "1", "--offset", "0", "--all", "--json"}, http.MethodGet, "/enterprise/enterprise-1/group/group-1/download/eventfeed/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "group-eventfeed-list-success.json", "group-eventfeed-list-success.golden", http.StatusOK, true, false},
		{"group report get", []string{"group-report", "get", "--start-date", "2026-01-01T00:00:00Z", "--end-date", "2026-01-31T23:59:59Z", "--json"}, http.MethodGet, "/enterprise/report/group-report/", url.Values{"start_date": {"2026-01-01T00:00:00Z"}, "end_date": {"2026-01-31T23:59:59Z"}}, "", "group-report-get-success.json", "group-report-get-success.golden", http.StatusOK, false, false},
		{"group thumbnail upload", []string{"group-thumbnail", "upload", "enterprise-1", "--image-file", image, "--json"}, http.MethodPost, "/enterprise/enterprise-1/devicegroup/thumbnail/", nil, "", "group-thumbnail-upload-success.json", "group-thumbnail-upload-success.golden", http.StatusOK, false, true},
		{"group thumbnail list", []string{"group-thumbnail", "list", "--enterprise", "enterprise-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/thumbnail/", nil, "", "group-thumbnail-list-success.json", "group-thumbnail-list-success.golden", http.StatusOK, false, false},
		{"group thumbnail get", []string{"group-thumbnail", "get", "enterprise-1", "thumbnail-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/thumbnail/thumbnail-1/", nil, "", "group-thumbnail-get-success.json", "group-thumbnail-get-success.golden", http.StatusOK, false, false},
		{"group thumbnail delete", []string{"group-thumbnail", "delete", "enterprise-1", "thumbnail-1", "--yes", "--json"}, http.MethodDelete, "/enterprise/enterprise-1/devicegroup/thumbnail/thumbnail-1/", nil, "", "group-thumbnail-delete-success.json", "group-thumbnail-delete-success.golden", http.StatusNoContent, false, false},
		{"legacy device group command get", []string{"legacy-device-group-command", "get", "enterprise-1", "group-1", "command-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/group-1/command/command-1/", nil, "", "legacy-device-group-command-get-success.json", "legacy-device-group-command-get-success.golden", http.StatusOK, false, false},
		{"sub group list", []string{"sub-group", "list", "--parent-group-ids", "group-1,group-2", "--immediate", "true", "--json"}, http.MethodGet, "/api/v2/subgroups/", url.Values{"parent_group_ids": {"group-1,group-2"}, "immediate": {"true"}}, "", "sub-group-list-success.json", "sub-group-list-success.golden", http.StatusOK, false, false},
		{"target list device group list", []string{"device-group", "list", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--json"}, http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/devicegroups/", nil, "", "target-list-device-group-list-success.json", "target-list-device-group-list-success.golden", http.StatusOK, false, false},
		{"target list device group add", []string{"device-group", "add", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--device-group-id", "group-1", "--json"}, http.MethodPost, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/devicegroups/", nil, `{"device_group_id":"group-1"}`, "target-list-device-group-add-success.json", "target-list-device-group-add-success.golden", http.StatusOK, false, false},
		{"target list device group delete", []string{"device-group", "delete", "group-1", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--yes", "--json"}, http.MethodDelete, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/devicegroups/group-1/", nil, "", "target-list-device-group-delete-success.json", "target-list-device-group-delete-success.golden", http.StatusOK, false, false},
		{"user group list", []string{"group", "list", "--user", "user-1", "--sub-groups", "true", "--json"}, http.MethodGet, "/authz2/v1/users/user-1/groups", url.Values{"sub_groups": {"true"}}, "", "user-group-list-success.json", "user-group-list-success.golden", http.StatusOK, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { executeGroupFixture(t, test) })
	}
}

func TestGroupCommandsAPIErrors(t *testing.T) {
	image := filepath.Join(t.TempDir(), "thumbnail.png")
	if err := os.WriteFile(image, []byte("group thumbnail fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []groupFixtureTest{
		{"device group list", []string{"device-group", "list", "--enterprise", "enterprise-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/", nil, "", "device-group-list-api-error.json", "", http.StatusBadRequest, false, false},
		{"device group create", []string{"device-group", "create", "--enterprise", "enterprise-1", "--name", "Fixture Group", "--json"}, http.MethodPost, "/enterprise/enterprise-1/devicegroup/", nil, `{"name":"Fixture Group"}`, "device-group-create-api-error.json", "", http.StatusBadRequest, false, false},
		{"group get", []string{"group", "get", "group-1", "enterprise-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/group-1/", nil, "", "group-get-api-error.json", "", http.StatusNotFound, false, false},
		{"group update", []string{"group", "update", "group-1", "enterprise-1", "--name", "Renamed Group", "--json"}, http.MethodPut, "/enterprise/enterprise-1/devicegroup/group-1/", nil, `{"name":"Renamed Group"}`, "group-update-api-error.json", "", http.StatusBadRequest, false, false},
		{"group delete", []string{"group", "delete", "group-1", "enterprise-1", "--yes", "--json"}, http.MethodDelete, "/enterprise/enterprise-1/devicegroup/group-1/", nil, "", "group-delete-api-error.json", "", http.StatusNotFound, false, false},
		{"group partial update", []string{"group", "partial-update", "group-1", "enterprise-1", "--name", "Partial Group", "--json"}, http.MethodPatch, "/enterprise/enterprise-1/devicegroup/group-1/", nil, `{"name":"Partial Group"}`, "group-partial-update-api-error.json", "", http.StatusBadRequest, false, false},
		{"group eventfeed list", []string{"group-eventfeed", "list", "--enterprise", "enterprise-1", "--group", "group-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/group/group-1/download/eventfeed/", nil, "", "group-eventfeed-list-api-error.json", "", http.StatusBadRequest, false, false},
		{"group report get", []string{"group-report", "get", "--start-date", "2026-01-01T00:00:00Z", "--end-date", "2026-01-31T23:59:59Z", "--json"}, http.MethodGet, "/enterprise/report/group-report/", url.Values{"start_date": {"2026-01-01T00:00:00Z"}, "end_date": {"2026-01-31T23:59:59Z"}}, "", "group-report-get-api-error.json", "", http.StatusBadRequest, false, false},
		{"group thumbnail upload", []string{"group-thumbnail", "upload", "enterprise-1", "--image-file", image, "--json"}, http.MethodPost, "/enterprise/enterprise-1/devicegroup/thumbnail/", nil, "", "group-thumbnail-upload-api-error.json", "", http.StatusBadRequest, false, true},
		{"group thumbnail list", []string{"group-thumbnail", "list", "--enterprise", "enterprise-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/thumbnail/", nil, "", "group-thumbnail-list-api-error.json", "", http.StatusBadRequest, false, false},
		{"group thumbnail get", []string{"group-thumbnail", "get", "enterprise-1", "thumbnail-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/thumbnail/thumbnail-1/", nil, "", "group-thumbnail-get-api-error.json", "", http.StatusNotFound, false, false},
		{"group thumbnail delete", []string{"group-thumbnail", "delete", "enterprise-1", "thumbnail-1", "--yes", "--json"}, http.MethodDelete, "/enterprise/enterprise-1/devicegroup/thumbnail/thumbnail-1/", nil, "", "group-thumbnail-delete-api-error.json", "", http.StatusNotFound, false, false},
		{"legacy device group command get", []string{"legacy-device-group-command", "get", "enterprise-1", "group-1", "command-1", "--json"}, http.MethodGet, "/enterprise/enterprise-1/devicegroup/group-1/command/command-1/", nil, "", "legacy-device-group-command-get-api-error.json", "", http.StatusNotFound, false, false},
		{"sub group list", []string{"sub-group", "list", "--json"}, http.MethodGet, "/api/v2/subgroups/", nil, "", "sub-group-list-api-error.json", "", http.StatusBadRequest, false, false},
		{"target list device group list", []string{"device-group", "list", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--json"}, http.MethodGet, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/devicegroups/", nil, "", "target-list-device-group-list-api-error.json", "", http.StatusUnauthorized, false, false},
		{"target list device group add", []string{"device-group", "add", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--device-group-id", "group-1", "--json"}, http.MethodPost, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/devicegroups/", nil, `{"device_group_id":"group-1"}`, "target-list-device-group-add-api-error.json", "", http.StatusUnauthorized, false, false},
		{"target list device group delete", []string{"device-group", "delete", "group-1", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--yes", "--json"}, http.MethodDelete, "/pipelines/v0/pipelines/pipeline-1/targetlists/target-list-1/devicegroups/group-1/", nil, "", "target-list-device-group-delete-api-error.json", "", http.StatusUnauthorized, false, false},
		{"user group list", []string{"group", "list", "--user", "user-1", "--json"}, http.MethodGet, "/authz2/v1/users/user-1/groups", nil, "", "user-group-list-api-error.json", "", http.StatusBadRequest, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { executeGroupFixture(t, test) })
	}
}

func TestGroupCommandInputValidation(t *testing.T) {
	for _, arguments := range [][]string{
		{"device-group", "create", "--enterprise", "enterprise-1"},
		{"group", "update", "group-1", "enterprise-1"},
		{"group", "partial-update", "group-1", "enterprise-1"},
		{"device-group", "create", "--enterprise", "enterprise-1", "--body", `{}`, "--name", "Fixture Group"},
	} {
		command := NewRootCommand()
		command.SetArgs(arguments)
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) error = %v", arguments, err)
		}
	}
}

func TestGroupDestructiveCommandsRequireConfirmation(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")

	for _, arguments := range [][]string{
		{"group", "delete", "group-1", "enterprise-1", "--json"},
		{"group-thumbnail", "delete", "enterprise-1", "thumbnail-1", "--json"},
		{"device-group", "delete", "group-1", "--pipeline", "pipeline-1", "--target-list", "target-list-1", "--json"},
	} {
		command := NewRootCommand()
		command.SetIn(bytes.NewBufferString("no\n"))
		command.SetArgs(arguments)
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) error = %v", arguments, err)
		}
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

type groupFixtureTest struct {
	name                  string
	arguments             []string
	method, path          string
	query                 url.Values
	body, fixture, golden string
	status                int
	all, multipart        bool
}

func executeGroupFixture(t *testing.T, test groupFixtureTest) {
	t.Helper()
	response := readGroupFixture(t, test.fixture)
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != test.method || request.URL.Path != test.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, test.method, test.path)
		}
		if test.all && requests == 2 {
			if request.URL.Query().Get("offset") != "1" {
				t.Errorf("pagination offset = %q, want 1", request.URL.Query().Get("offset"))
			}
			response = readGroupFixture(t, test.fixture[:len(test.fixture)-len("success.json")]+"second-page.json")
		} else if request.URL.Query().Encode() != test.query.Encode() {
			t.Errorf("request query = %q, want %q", request.URL.Query(), test.query)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		if test.multipart {
			mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
			if err != nil || mediaType != "multipart/form-data" {
				t.Errorf("content type = %q", request.Header.Get("Content-Type"))
			}
			if err := request.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			file, err := request.MultipartForm.File["image_file"][0].Open()
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(file)
			file.Close()
			if string(data) != "group thumbnail fixture" {
				t.Errorf("image_file = %q", data)
			}
		} else {
			data, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != test.body {
				t.Errorf("request body = %q, want %q", data, test.body)
			}
			if test.body != "" && request.Header.Get("Content-Type") != "application/json" {
				t.Errorf("content type = %q", request.Header.Get("Content-Type"))
			}
		}
		if test.all && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+test.path+"?offset=1"), 1)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(test.status)
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
	if test.status >= http.StatusBadRequest {
		var apiError *esperruntime.APIError
		if !errors.As(err, &apiError) || apiError.StatusCode != test.status || esperruntime.ExitCode(err) != 1 {
			t.Fatalf("Execute() error = %v", err)
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
	if err := json.Unmarshal(readGroupFixture(t, test.golden), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", string(readGroupFixture(t, test.golden)), output.String())
	}
}

func readGroupFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "group", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
