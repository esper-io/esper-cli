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
	"reflect"
	"strings"
	"testing"

	"github.com/esper-io/esper-cli/internal/cmd/generated"
	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

type apnsFixture struct {
	key, fixture, method, path, body string
	args                             []string
	query                            url.Values
	status, errorStatus              int
	all, destructive, raw, multipart bool
}

func apnsFixtures() []apnsFixture {
	return []apnsFixture{
		{"v0 GET /tenant/v0/apnscertificates/", "apns-cert-list", http.MethodGet, "/tenant/v0/apnscertificates/", "", []string{"apns-cert", "list", "--state", "CSR_GENERATED", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"state": {"CSR_GENERATED"}, "limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusBadRequest, true, false, false, false},
		{"v0 POST /tenant/v0/apnscertificates/", "apns-csr-list", http.MethodPost, "/tenant/v0/apnscertificates/", "{}", []string{"apns-csr", "list"}, nil, http.StatusOK, http.StatusBadRequest, false, false, true, false},
		{"v0 GET /tenant/v0/apnscertificates/{id}/", "apns-cert-get", http.MethodGet, "/tenant/v0/apnscertificates/cert-1/", "", []string{"apns-cert", "get", "cert-1", "--json"}, nil, http.StatusOK, http.StatusBadRequest, false, false, false, false},
		{"v0 PUT /tenant/v0/apnscertificates/{id}/", "apns-cert-upload", http.MethodPut, "/tenant/v0/apnscertificates/cert-1/", "", []string{"apns-cert", "upload", "cert-1", "--file", "FIXTURE_FILE", "--apple-id", "apple@example.test", "--json"}, nil, http.StatusOK, http.StatusBadRequest, false, false, false, true},
		{"v0 DELETE /tenant/v0/apnscertificates/{id}/", "apns-cert-delete", http.MethodDelete, "/tenant/v0/apnscertificates/cert-1/", "", []string{"apns-cert", "delete", "cert-1", "--yes", "--json"}, nil, http.StatusOK, http.StatusBadRequest, false, true, false, false},
	}
}

func TestAPNsOperationCoverage(t *testing.T) {
	rows, actual := map[string]bool{}, map[string]bool{}
	for _, row := range apnsFixtures() {
		rows[row.key] = true
	}
	if len(rows) != 5 {
		t.Fatalf("fixture rows = %d, want 5", len(rows))
	}
	for _, operation := range generated.Operations() {
		if operation.Noun == "apns-cert" || operation.Noun == "apns-csr" {
			actual[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if !reflect.DeepEqual(rows, actual) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(rows), len(actual))
	}
}

func TestAPNsCommandsGoldenFixtures(t *testing.T) {
	for _, row := range apnsFixtures() {
		t.Run(row.fixture, func(t *testing.T) { executeAPNsFixture(t, row, false) })
	}
}

func TestAPNsCommandsAPIErrors(t *testing.T) {
	for _, row := range apnsFixtures() {
		t.Run(row.fixture, func(t *testing.T) { executeAPNsFixture(t, row, true) })
	}
}

func TestAPNsInputValidation(t *testing.T) {
	for _, args := range [][]string{
		{"apns-cert", "get"},
		{"apns-cert", "upload"},
		{"apns-cert", "upload", "cert-1", "--file", "missing.pem"},
		{"apns-cert", "upload", "cert-1", "--apple-id", "apple@example.test"},
		{"apns-cert", "delete"},
	} {
		command := NewRootCommand()
		command.SetArgs(args)
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) = %v", args, err)
		}
	}
}

func TestAPNsRawOutputAndDeclinedDelete(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.URL.Path == "/tenant/v0/apnscertificates/" {
			_, _ = writer.Write(readAPNsFixture(t, "apns-csr-list-success.json"))
		}
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")

	command := NewRootCommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetArgs([]string{"apns-csr", "list"})
	if err := command.Execute(); err != nil || !bytes.Equal(stdout.Bytes(), readAPNsFixture(t, "apns-csr-list-success.golden")) {
		t.Fatalf("raw stdout = %q, error = %v", stdout.String(), err)
	}

	output := filepath.Join(t.TempDir(), "apns.csr")
	command = NewRootCommand()
	command.SetArgs([]string{"apns-csr", "list", "--output", output})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(output); err != nil || !bytes.Equal(data, readAPNsFixture(t, "apns-csr-list-success.golden")) {
		t.Fatalf("raw output file = %q, error = %v", data, err)
	}

	command = NewRootCommand()
	command.SetArgs([]string{"apns-csr", "list", "--json"})
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("raw --json error = %v", err)
	}

	command = NewRootCommand()
	command.SetIn(strings.NewReader("no\n"))
	command.SetArgs([]string{"apns-cert", "delete", "cert-1"})
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("declined delete error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func executeAPNsFixture(t *testing.T, row apnsFixture, apiError bool) {
	t.Helper()
	fixture, status := row.fixture+"-success.json", row.status
	if apiError {
		fixture, status = row.fixture+"-api-error.json", row.errorStatus
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != row.method || request.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, row.method, row.path)
		}
		if got, want := request.URL.Query().Encode(), row.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		accept := "application/json"
		if row.raw {
			accept = "text/plain; charset=utf-8"
		}
		if request.Header.Get("Accept") != accept {
			t.Errorf("accept = %q, want %q", request.Header.Get("Accept"), accept)
		}
		assertAPNsRequest(t, request, row)
		if !row.raw {
			writer.Header().Set("Content-Type", "application/json")
		}
		writer.WriteHeader(status)
		_, _ = writer.Write(readAPNsFixture(t, fixture))
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")

	args := append([]string(nil), row.args...)
	for index, argument := range args {
		if argument == "FIXTURE_FILE" {
			file := filepath.Join(t.TempDir(), "apns.pem")
			if err := os.WriteFile(file, []byte("APNS certificate fixture"), 0o600); err != nil {
				t.Fatal(err)
			}
			args[index] = file
		}
	}
	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(args)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != status || esperruntime.ExitCode(err) != 1 {
			t.Fatalf("API error = %v", err)
		}
		if want := readAPNsFixture(t, fixture); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %q, want %q", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if row.all && requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	if want := readAPNsFixture(t, row.fixture+"-success.golden"); !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.Bytes(), want)
	}
}

func assertAPNsRequest(t *testing.T, request *http.Request, row apnsFixture) {
	t.Helper()
	if row.multipart {
		mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Fatalf("content type = %q", request.Header.Get("Content-Type"))
		}
		if err := request.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if values := request.MultipartForm.Value["apple_id"]; !reflect.DeepEqual(values, []string{"apple@example.test"}) {
			t.Fatalf("apple_id = %q", values)
		}
		files := request.MultipartForm.File["file"]
		if len(files) != 1 {
			t.Fatalf("file count = %d, want 1", len(files))
		}
		file, err := files[0].Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil || !bytes.Equal(data, []byte("APNS certificate fixture")) {
			t.Fatalf("file bytes = %q, error = %v", data, err)
		}
		return
	}
	data, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != row.body {
		t.Errorf("body = %q, want %q", data, row.body)
	}
	if row.body != "" && request.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type = %q, want application/json", request.Header.Get("Content-Type"))
	}
}

func readAPNsFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "apns", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
