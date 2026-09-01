package cmd

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestConvergeCommandsGoldenFixtures(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		method    string
		path      string
		query     url.Values
		body      string
		fixture   string
		golden    string
		status    int
		all       bool
	}{
		{name: "converge list success", arguments: []string{"converge", "list", "--device-ids", "device-1,device-2", "--limit", "1", "--offset", "0", "--all", "--json"}, method: http.MethodGet, path: "/v2/converge", query: url.Values{"device_ids": {"device-1,device-2"}, "limit": {"1"}, "offset": {"0"}}, fixture: "converge-list-success.json", golden: "converge-list-success.golden", status: http.StatusOK, all: true},
		{name: "converge create success", arguments: []string{"converge", "create", "--device-id", "device-1", "--converge-with-provision-option", "true", "--schedule-type", "IMMEDIATE", "--json"}, method: http.MethodPost, path: "/v2/converge", body: `{"converge_with_provision_option":true,"device_id":"device-1","schedule_type":"IMMEDIATE"}`, fixture: "converge-create-success.json", golden: "converge-create-success.golden", status: http.StatusCreated},
		{name: "converge get success", arguments: []string{"converge", "get", "converge-1", "--json"}, method: http.MethodGet, path: "/v2/converge/converge-1", fixture: "converge-get-success.json", golden: "converge-get-success.golden", status: http.StatusOK},
		{name: "converge list API error", arguments: []string{"converge", "list", "--json"}, method: http.MethodGet, path: "/v2/converge", fixture: "converge-list-api-error.json", status: http.StatusUnauthorized},
		{name: "converge create API error", arguments: []string{"converge", "create", "--device-id", "device-1", "--converge-with-provision-option", "true", "--json"}, method: http.MethodPost, path: "/v2/converge", body: `{"converge_with_provision_option":true,"device_id":"device-1"}`, fixture: "converge-create-api-error.json", status: http.StatusBadRequest},
		{name: "converge get API error", arguments: []string{"converge", "get", "converge-1", "--json"}, method: http.MethodGet, path: "/v2/converge/converge-1", fixture: "converge-get-api-error.json", status: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executeConvergeFixture(t, test.arguments, test.method, test.path, test.query, test.body, test.fixture, test.golden, test.status, test.all)
		})
	}
}

func TestConvergeCommandInputValidation(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
	}{
		{name: "converge create requires device ID", arguments: []string{"converge", "create", "--converge-with-provision-option", "true"}},
		{name: "converge create requires provision option", arguments: []string{"converge", "create", "--device-id", "device-1"}},
		{name: "converge create body cannot combine property flags", arguments: []string{"converge", "create", "--body", `{}`, "--device-id", "device-1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := NewRootCommand()
			command.SetArgs(test.arguments)
			err := command.Execute()
			if err == nil || esperruntime.ExitCode(err) != 2 {
				t.Fatalf("Execute() error = %v", err)
			}
		})
	}
}

func executeConvergeFixture(t *testing.T, arguments []string, method, path string, query url.Values, body, fixture, golden string, status int, all bool) {
	t.Helper()
	response := readConvergeFixture(t, fixture)
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != method {
			t.Errorf("request method = %q, want %q", request.Method, method)
		}
		if request.URL.Path != path {
			t.Errorf("request path = %q, want %q", request.URL.Path, path)
		}
		if all && requests == 2 {
			if request.URL.Query().Get("offset") != "1" {
				t.Errorf("pagination offset = %q, want 1", request.URL.Query().Get("offset"))
			}
			response = readConvergeFixture(t, "converge-list-second-page.json")
		} else if request.URL.Query().Encode() != query.Encode() {
			t.Errorf("request query = %q, want %q", request.URL.Query(), query)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		data, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != body {
			t.Errorf("request body = %q, want %q", data, body)
		}
		if body != "" && request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q, want application/json", request.Header.Get("Content-Type"))
		}
		if all && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+path+"?offset=1"), 1)
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
	command.SetArgs(arguments)
	err := command.Execute()
	if status >= http.StatusBadRequest {
		var apiError *esperruntime.APIError
		if !errors.As(err, &apiError) || apiError.StatusCode != status {
			t.Fatalf("Execute() error = %v", err)
		}
		if exitCode := esperruntime.ExitCode(err); exitCode != 1 {
			t.Fatalf("ExitCode() = %d, want 1", exitCode)
		}
		return
	}
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if all && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if want := string(readConvergeFixture(t, golden)); output.String() != want {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", want, output.String())
	}
}

func readConvergeFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "spec", "fixtures", "converge", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
