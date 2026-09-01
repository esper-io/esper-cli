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
	"strings"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestProvisioningProfileCommandsGoldenFixtures(t *testing.T) {
	file := writeProvisioningProfileFile(t)
	tests := []struct {
		name, fixture, golden, method, path string
		arguments                           []string
		input                               string
		query                               url.Values
		multipart                           bool
		all                                 bool
	}{
		{"create", "provisioning-profile-create-success.json", "provisioning-profile-create-success.golden", http.MethodPost, "/v2/provisioning-profiles", []string{"provisioning-profile", "create", "--file", file, "--json"}, "", nil, true, false},
		{"list", "provisioning-profile-list-success.json", "provisioning-profile-list-success.golden", http.MethodGet, "/v2/provisioning-profiles", []string{"provisioning-profile", "list", "--profile-type", "enterprise", "--limit", "1", "--offset", "0", "--all", "--json"}, "", url.Values{"profile_type": {"enterprise"}, "limit": {"1"}, "offset": {"0"}}, false, true},
		{"get", "provisioning-profile-get-success.json", "provisioning-profile-get-success.golden", http.MethodGet, "/v2/provisioning-profiles/profile-1", []string{"provisioning-profile", "get", "profile-1", "--json"}, "", nil, false, false},
		{"version get", "provisioning-profile-version-get-success.json", "provisioning-profile-version-get-success.golden", http.MethodGet, "/v2/provisioning-profiles/profile-1/versions/version-1", []string{"provisioning-profile-version", "get", "version-1", "--provisioning-profile", "profile-1", "--json"}, "", nil, false, false},
		{"version delete", "provisioning-profile-version-delete-success.json", "provisioning-profile-version-delete-success.golden", http.MethodDelete, "/v2/provisioning-profiles/profile-1/versions/version-1", []string{"provisioning-profile-version", "delete", "version-1", "--provisioning-profile", "profile-1", "--json"}, "yes\n", nil, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executeProvisioningProfileFixture(t, test.arguments, test.input, test.method, test.path, test.query, test.fixture, test.golden, http.StatusOK, test.multipart, test.all)
		})
	}
}

func TestProvisioningProfileCommandsAPIErrors(t *testing.T) {
	file := writeProvisioningProfileFile(t)
	tests := []struct {
		arguments             []string
		path, fixture, method string
		multipart             bool
	}{
		{[]string{"provisioning-profile", "create", "--file", file, "--json"}, "/v2/provisioning-profiles", "provisioning-profile-create-api-error.json", http.MethodPost, true},
		{[]string{"provisioning-profile", "list", "--json"}, "/v2/provisioning-profiles", "provisioning-profile-list-api-error.json", http.MethodGet, false},
		{[]string{"provisioning-profile", "get", "profile-1", "--json"}, "/v2/provisioning-profiles/profile-1", "provisioning-profile-get-api-error.json", http.MethodGet, false},
		{[]string{"provisioning-profile-version", "get", "version-1", "--provisioning-profile", "profile-1", "--json"}, "/v2/provisioning-profiles/profile-1/versions/version-1", "provisioning-profile-version-get-api-error.json", http.MethodGet, false},
		{[]string{"provisioning-profile-version", "delete", "version-1", "--provisioning-profile", "profile-1", "--yes", "--json"}, "/v2/provisioning-profiles/profile-1/versions/version-1", "provisioning-profile-version-delete-api-error.json", http.MethodDelete, false},
	}
	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			executeProvisioningProfileFixture(t, test.arguments, "", test.method, test.path, nil, test.fixture, "", http.StatusBadRequest, test.multipart, false)
		})
	}
}

func TestProvisioningProfileInputValidation(t *testing.T) {
	for _, arguments := range [][]string{
		{"provisioning-profile", "create"},
		{"provisioning-profile-version", "get", "version-1"},
		{"provisioning-profile-version", "delete", "version-1"},
	} {
		command := NewRootCommand()
		command.SetArgs(arguments)
		err := command.Execute()
		if err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) error = %v", arguments, err)
		}
	}
}

func executeProvisioningProfileFixture(t *testing.T, arguments []string, input, method, path string, query url.Values, fixture, golden string, status int, multipart, all bool) {
	t.Helper()
	response := readProvisioningProfileFixture(t, fixture)
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != method || request.URL.Path != path || request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("request = %s %s authorization %q", request.Method, request.URL.Path, request.Header.Get("Authorization"))
		}
		if all && requests == 2 {
			if request.URL.Query().Get("offset") != "1" {
				t.Errorf("pagination offset = %q", request.URL.Query().Get("offset"))
			}
			response = readProvisioningProfileFixture(t, "provisioning-profile-list-second-page.json")
		} else if request.URL.Query().Encode() != query.Encode() {
			t.Errorf("request query = %q, want %q", request.URL.Query(), query)
		}
		if multipart {
			mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
			if err != nil || mediaType != "multipart/form-data" {
				t.Errorf("content type = %q", request.Header.Get("Content-Type"))
			}
			if err := request.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			file, err := request.MultipartForm.File["file"][0].Open()
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(file)
			file.Close()
			if string(data) != "provisioning profile fixture" {
				t.Errorf("file = %q", data)
			}
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
	var stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetIn(strings.NewReader(input))
	command.SetArgs(arguments)
	err := command.Execute()
	if status >= http.StatusBadRequest {
		var apiError *esperruntime.APIError
		if !errors.As(err, &apiError) || apiError.StatusCode != status || esperruntime.ExitCode(err) != 1 {
			t.Fatalf("Execute() error = %v", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if all && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	var got, want any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(readProvisioningProfileFixture(t, golden), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("output mismatch\n%s", output.String())
	}
}

func writeProvisioningProfileFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.mobileprovision")
	if err := os.WriteFile(path, []byte("provisioning profile fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readProvisioningProfileFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "provisioning-profile", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
