package cmd

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestContentCommandsGoldenFixtures(t *testing.T) {
	file := writeContentUploadFile(t)
	tests := []struct {
		name      string
		arguments []string
		method    string
		path      string
		query     url.Values
		body      string
		multipart map[string]string
		fixture   string
		golden    string
		status    int
		all       bool
	}{
		{name: "content list success", arguments: []string{"content", "list", "--enterprise", "enterprise-1", "--search", "guide", "--limit", "1", "--offset", "0", "--all", "--json"}, method: http.MethodGet, path: "/v0/enterprise/enterprise-1/content/", query: url.Values{"search": {"guide"}, "limit": {"1"}, "offset": {"0"}}, fixture: "content-list-success.json", golden: "content-list-success.golden", status: http.StatusOK, all: true},
		{name: "content get success", arguments: []string{"content", "get", "enterprise-1", "content-1", "--json"}, method: http.MethodGet, path: "/v0/enterprise/enterprise-1/content/content-1/", fixture: "content-get-success.json", golden: "content-get-success.golden", status: http.StatusOK},
		{name: "content delete success", arguments: []string{"content", "delete", "enterprise-1", "content-1", "--yes", "--json"}, method: http.MethodDelete, path: "/v0/enterprise/enterprise-1/content/content-1/", fixture: "content-delete-success.body", golden: "content-delete-success.golden", status: http.StatusNoContent},
		{name: "content patch success", arguments: []string{"content", "patch", "enterprise-1", "content-1", "--description", "Updated fixture", "--json"}, method: http.MethodPatch, path: "/v0/enterprise/enterprise-1/content/content-1/", body: `{"description":"Updated fixture"}`, fixture: "content-patch-success.json", golden: "content-patch-success.golden", status: http.StatusOK},
		{name: "content create success", arguments: []string{"content", "create", "--enterprise", "enterprise-1", "--key", file, "--json"}, method: http.MethodPost, path: "/v0/enterprise/enterprise-1/content/upload/", multipart: map[string]string{"key": "content fixture"}, fixture: "content-create-success.json", golden: "content-create-success.golden", status: http.StatusOK},
		{name: "remote file upload success", arguments: []string{"remote-file", "upload", "enterprise-1", "--file", file, "--filename", "guide.pdf", "--content-type", "application/pdf", "--json"}, method: http.MethodPost, path: "/v0/enterprise/enterprise-1/content/remote-file/", multipart: map[string]string{"file": "content fixture", "filename": "guide.pdf", "content_type": "application/pdf"}, fixture: "remote-file-upload-success.json", golden: "remote-file-upload-success.golden", status: http.StatusCreated},
		{name: "download generate success", arguments: []string{"download", "generate", "enterprise-1", "--file-key", "enterprise-1/files/guide.pdf", "--expires-in", "600", "--json"}, method: http.MethodPost, path: "/v0/enterprise/enterprise-1/content/remote-file/generate_download_url/", body: `{"expires_in":600,"file_key":"enterprise-1/files/guide.pdf"}`, fixture: "download-generate-success.json", golden: "download-generate-success.golden", status: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executeContentFixture(t, test.arguments, test.method, test.path, test.query, test.body, test.multipart, test.fixture, test.golden, test.status, test.all)
		})
	}
}

func TestContentCommandsAPIErrors(t *testing.T) {
	file := writeContentUploadFile(t)
	tests := []struct {
		name      string
		arguments []string
		method    string
		path      string
		body      string
		multipart map[string]string
		fixture   string
	}{
		{name: "content list API error", arguments: []string{"content", "list", "--enterprise", "enterprise-1", "--json"}, method: http.MethodGet, path: "/v0/enterprise/enterprise-1/content/", fixture: "content-list-api-error.json"},
		{name: "content get API error", arguments: []string{"content", "get", "enterprise-1", "content-1", "--json"}, method: http.MethodGet, path: "/v0/enterprise/enterprise-1/content/content-1/", fixture: "content-get-api-error.json"},
		{name: "content delete API error", arguments: []string{"content", "delete", "enterprise-1", "content-1", "--yes", "--json"}, method: http.MethodDelete, path: "/v0/enterprise/enterprise-1/content/content-1/", fixture: "content-delete-api-error.json"},
		{name: "content patch API error", arguments: []string{"content", "patch", "enterprise-1", "content-1", "--description", "Updated fixture", "--json"}, method: http.MethodPatch, path: "/v0/enterprise/enterprise-1/content/content-1/", body: `{"description":"Updated fixture"}`, fixture: "content-patch-api-error.json"},
		{name: "content create API error", arguments: []string{"content", "create", "--enterprise", "enterprise-1", "--key", file, "--json"}, method: http.MethodPost, path: "/v0/enterprise/enterprise-1/content/upload/", multipart: map[string]string{"key": "content fixture"}, fixture: "content-create-api-error.json"},
		{name: "remote file upload API error", arguments: []string{"remote-file", "upload", "enterprise-1", "--file", file, "--json"}, method: http.MethodPost, path: "/v0/enterprise/enterprise-1/content/remote-file/", multipart: map[string]string{"file": "content fixture"}, fixture: "remote-file-upload-api-error.json"},
		{name: "download generate API error", arguments: []string{"download", "generate", "enterprise-1", "--file-key", "enterprise-1/files/guide.pdf", "--json"}, method: http.MethodPost, path: "/v0/enterprise/enterprise-1/content/remote-file/generate_download_url/", body: `{"file_key":"enterprise-1/files/guide.pdf"}`, fixture: "download-generate-api-error.json"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executeContentFixture(t, test.arguments, test.method, test.path, nil, test.body, test.multipart, test.fixture, "", http.StatusBadRequest, false)
		})
	}
}

func TestContentCommandInputValidation(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
	}{
		{name: "content create requires key", arguments: []string{"content", "create", "--enterprise", "enterprise-1"}},
		{name: "remote file upload requires file", arguments: []string{"remote-file", "upload", "enterprise-1"}},
		{name: "download generate requires file key", arguments: []string{"download", "generate", "enterprise-1"}},
		{name: "content patch body cannot combine property flags", arguments: []string{"content", "patch", "enterprise-1", "content-1", "--body", `{}`, "--description", "Updated fixture"}},
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

func executeContentFixture(t *testing.T, arguments []string, method, path string, query url.Values, body string, multipartValues map[string]string, fixture, golden string, status int, all bool) {
	t.Helper()
	response := readContentFixture(t, fixture)
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
			response = readContentFixture(t, "content-list-second-page.json")
		} else if request.URL.Query().Encode() != query.Encode() {
			t.Errorf("request query = %q, want %q", request.URL.Query(), query)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		validateContentRequest(t, request, body, multipartValues)
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
	if want := string(readContentFixture(t, golden)); output.String() != want {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", want, output.String())
	}
}

func validateContentRequest(t *testing.T, request *http.Request, body string, values map[string]string) {
	t.Helper()
	if len(values) == 0 {
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
		return
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("content type = %q, want multipart/form-data", request.Header.Get("Content-Type"))
	}
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	for name, want := range values {
		if fileHeaders := request.MultipartForm.File[name]; len(fileHeaders) > 0 {
			file, err := fileHeaders[0].Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(file)
			file.Close()
			if err != nil || string(data) != want {
				t.Errorf("multipart file %s = %q, want %q", name, data, want)
			}
			continue
		}
		if got := request.MultipartForm.Value[name]; len(got) != 1 || got[0] != want {
			t.Errorf("multipart field %s = %q, want %q", name, got, want)
		}
	}
}

func writeContentUploadFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "guide.pdf")
	if err := os.WriteFile(path, []byte("content fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readContentFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "spec", "fixtures", "content", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
