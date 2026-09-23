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
	"strings"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestEnterpriseCommandsGoldenFixtures(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		input     string
		method    string
		path      string
		query     url.Values
		body      string
		fixture   string
		golden    string
		status    int
	}{
		{
			name:      "enterprise get success",
			arguments: []string{"enterprise", "get", "enterprise-1", "--json"},
			method:    http.MethodGet,
			path:      "/v1/enterprise/enterprise-1/",
			fixture:   "enterprise-get-success.json",
			golden:    "enterprise-get-success.golden",
			status:    http.StatusOK,
		},
		{
			name:      "enterprise get API error",
			arguments: []string{"enterprise", "get", "enterprise-1", "--json"},
			method:    http.MethodGet,
			path:      "/v1/enterprise/enterprise-1/",
			fixture:   "enterprise-get-api-error.json",
			status:    http.StatusNotFound,
		},
		{
			name:      "enterprise partial update success",
			arguments: []string{"enterprise", "partial-update", "enterprise-1", "--name", "Renamed Fixture Enterprise", "--json"},
			method:    http.MethodPatch,
			path:      "/v1/enterprise/enterprise-1/",
			body:      `{"name":"Renamed Fixture Enterprise"}`,
			fixture:   "enterprise-partial-update-success.json",
			golden:    "enterprise-partial-update-success.golden",
			status:    http.StatusOK,
		},
		{
			name:      "enterprise partial update API error",
			arguments: []string{"enterprise", "partial-update", "enterprise-1", "--name", "Renamed Fixture Enterprise", "--json"},
			method:    http.MethodPatch,
			path:      "/v1/enterprise/enterprise-1/",
			body:      `{"name":"Renamed Fixture Enterprise"}`,
			fixture:   "enterprise-partial-update-api-error.json",
			status:    http.StatusBadRequest,
		},
		{
			name:      "enterprise policy delete success with yes",
			arguments: []string{"enterprise-policy", "delete", "enterprise-1", "42", "--yes", "--json"},
			method:    http.MethodDelete,
			path:      "/enterprise/enterprise-1/policy/42/",
			fixture:   "enterprise-policy-delete-success.body",
			golden:    "enterprise-policy-delete-success.golden",
			status:    http.StatusNoContent,
		},
		{
			name:      "enterprise policy delete API error after confirmation",
			arguments: []string{"enterprise-policy", "delete", "enterprise-1", "42", "--json"},
			input:     "yes\n",
			method:    http.MethodDelete,
			path:      "/enterprise/enterprise-1/policy/42/",
			fixture:   "enterprise-policy-delete-api-error.json",
			status:    http.StatusNotFound,
		},
		{
			name:      "enterprise report get success",
			arguments: []string{"enterprise-report", "get", "--start-date", "2026-01-01T00:00:00Z", "--end-date", "2026-01-31T23:59:59Z", "--json"},
			method:    http.MethodGet,
			path:      "/enterprise/report/enterprise-report/",
			query:     url.Values{"start_date": {"2026-01-01T00:00:00Z"}, "end_date": {"2026-01-31T23:59:59Z"}},
			fixture:   "enterprise-report-get-success.json",
			golden:    "enterprise-report-get-success.golden",
			status:    http.StatusOK,
		},
		{
			name:      "enterprise report get API error",
			arguments: []string{"enterprise-report", "get", "--start-date", "2026-01-01T00:00:00Z", "--end-date", "2026-01-31T23:59:59Z", "--json"},
			method:    http.MethodGet,
			path:      "/enterprise/report/enterprise-report/",
			query:     url.Values{"start_date": {"2026-01-01T00:00:00Z"}, "end_date": {"2026-01-31T23:59:59Z"}},
			fixture:   "enterprise-report-get-api-error.json",
			status:    http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := readEnterpriseFixture(t, test.fixture)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != test.method {
					t.Errorf("request method = %q, want %q", request.Method, test.method)
				}
				if request.URL.Path != test.path {
					t.Errorf("request path = %q, want %q", request.URL.Path, test.path)
				}
				if request.URL.Query().Encode() != test.query.Encode() {
					t.Errorf("request query = %q, want %q", request.URL.Query(), test.query)
				}
				if request.Header.Get("Authorization") != "Bearer fixture-key" {
					t.Errorf("authorization = %q", request.Header.Get("Authorization"))
				}
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				if string(body) != test.body {
					t.Errorf("request body = %q, want %q", body, test.body)
				}
				if test.body != "" {
					if request.Header.Get("Content-Type") != "application/json" {
						t.Errorf("content type = %q", request.Header.Get("Content-Type"))
					}
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
			command.SetIn(strings.NewReader(test.input))
			command.SetOut(&output)
			command.SetErr(&output)
			command.SetArgs(test.arguments)
			err := command.Execute()
			if test.status >= http.StatusBadRequest {
				var apiError *esperruntime.APIError
				if !errors.As(err, &apiError) || apiError.StatusCode != test.status {
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
			want := string(readEnterpriseFixture(t, test.golden))
			if output.String() != want {
				t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", want, output.String())
			}
		})
	}
}

func readEnterpriseFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "spec", "fixtures", "enterprise", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
