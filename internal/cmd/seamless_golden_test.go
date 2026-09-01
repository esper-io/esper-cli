package cmd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/esper-io/esper-cli/internal/cmd/generated"
	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

type seamlessFixture struct {
	key, name, method, path, body string
	args                          []string
	status, errorStatus           int
	multipart                     bool
}

const seamlessCreateBody = `{"assigned_blueprint_id":"22222222-2222-4222-8222-222222222222","alias":"Warehouse scanner","tags":["warehouse","pilot"],"seamless_info":{"unique_ids":"550e8400-e29b-41d4-a716-446655440000","unique_id_type":3,"platform":"APPLE"}}`

func seamlessFixtures(t *testing.T) []seamlessFixture {
	t.Helper()
	csv := filepath.Join(t.TempDir(), "devices.csv")
	if err := os.WriteFile(csv, []byte("serial_number\nSERIAL-UPLOAD-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return []seamlessFixture{
		{"v2 POST /v2/seamless/upload", "seamless-upload", http.MethodPost, "/v2/seamless/upload", "", []string{"seamless", "upload", "--csv-file", csv, "--platform", "WINDOWS", "--json"}, http.StatusOK, http.StatusBadRequest, true},
		{"v2 POST /v2/seamless", "seamless-create", http.MethodPost, "/v2/seamless", seamlessCreateBody, []string{"seamless", "create", "--body", seamlessCreateBody, "--json"}, http.StatusOK, http.StatusBadRequest, false},
	}
}

func TestSeamlessOperationCoverage(t *testing.T) {
	expected, actual := map[string]bool{}, map[string]bool{}
	for _, row := range seamlessFixtures(t) {
		expected[row.key] = true
	}
	if len(expected) != 2 {
		t.Fatalf("fixture rows = %d, want 2", len(expected))
	}
	for _, operation := range generated.Operations() {
		if operation.Noun == "seamless" {
			actual[operation.Generation+" "+operation.Method+" "+operation.Path] = true
			if operation.Pagination != "none" {
				t.Fatalf("%s pagination = %q", operation.Path, operation.Pagination)
			}
		}
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("operation keys = %#v, want %#v", actual, expected)
	}
}

func TestSeamlessCommandsGoldenFixtures(t *testing.T) {
	for _, row := range seamlessFixtures(t) {
		t.Run(row.name, func(t *testing.T) { executeSeamlessFixture(t, row, false) })
	}
}

func TestSeamlessCommandsAPIErrors(t *testing.T) {
	for _, row := range seamlessFixtures(t) {
		t.Run(row.name, func(t *testing.T) { executeSeamlessFixture(t, row, true) })
	}
}

func TestSeamlessFixturesMatchResponseContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "v2.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, row := range seamlessFixtures(t) {
		parts := strings.SplitN(row.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		for status, name := range map[int]string{row.status: row.name + "-success.json", row.errorStatus: row.name + "-api-error.json"} {
			var value any
			if err := json.Unmarshal(readSeamlessFixture(t, name), &value); err != nil {
				t.Fatal(err)
			}
			if err := fixtureSchemaValidate(document, fixtureSchemaResponse(t, document, operation, status), value, "$", map[string]bool{}); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		}
	}
}

func TestSeamlessUUIDFormats(t *testing.T) {
	var request struct {
		AssignedBlueprintID string `json:"assigned_blueprint_id"`
		SeamlessInfo        struct {
			UniqueIDs    string `json:"unique_ids"`
			UniqueIDType int    `json:"unique_id_type"`
		} `json:"seamless_info"`
	}
	if err := json.Unmarshal([]byte(seamlessCreateBody), &request); err != nil {
		t.Fatal(err)
	}
	assertSeamlessUUID(t, request.AssignedBlueprintID)
	if request.SeamlessInfo.UniqueIDType == 3 {
		assertSeamlessUUID(t, request.SeamlessInfo.UniqueIDs)
	}
	for _, name := range []string{"seamless-create-success.json", "seamless-upload-success.json"} {
		var response struct {
			Content struct {
				Results []struct {
					ID                  string  `json:"id"`
					AssignedBlueprintID string  `json:"assigned_blueprint_id"`
					GroupID             *string `json:"group_id"`
					UniqueIDs           string  `json:"unique_ids"`
					UniqueIDType        string  `json:"unique_id_type"`
				} `json:"results"`
			} `json:"content"`
		}
		if err := json.Unmarshal(readSeamlessFixture(t, name), &response); err != nil {
			t.Fatal(err)
		}
		for _, result := range response.Content.Results {
			assertSeamlessUUID(t, result.ID)
			assertSeamlessUUID(t, result.AssignedBlueprintID)
			if result.GroupID != nil {
				assertSeamlessUUID(t, *result.GroupID)
			}
			if result.UniqueIDType == "UUID" {
				assertSeamlessUUID(t, result.UniqueIDs)
			}
		}
	}
}

func assertSeamlessUUID(t *testing.T, value string) {
	t.Helper()
	compact := strings.ReplaceAll(value, "-", "")
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' || len(compact) != 32 {
		t.Fatalf("%q is not a UUID", value)
	}
	if _, err := hex.DecodeString(compact); err != nil {
		t.Fatalf("%q is not a UUID: %v", value, err)
	}
}

func TestSeamlessInputRules(t *testing.T) {
	root := NewRootCommand()
	upload, _, err := root.Find([]string{"seamless", "upload"})
	if err != nil || upload.Flags().Lookup("csv-file") == nil || upload.Flags().Lookup("platform") == nil || upload.Flags().Lookup("body") != nil {
		t.Fatalf("upload flags: csv-file=%v platform=%v body=%v error=%v", upload.Flags().Lookup("csv-file"), upload.Flags().Lookup("platform"), upload.Flags().Lookup("body"), err)
	}
	create, _, err := root.Find([]string{"seamless", "create"})
	if err != nil || create.Flags().Lookup("body") == nil {
		t.Fatalf("create body flag = %v, error = %v", create.Flags().Lookup("body"), err)
	}
	for _, name := range []string{"assigned-blueprint-id", "group-id", "alias", "tags", "seamless-info", "limit", "offset", "all"} {
		if create.Flags().Lookup(name) != nil {
			t.Fatalf("create exposes --%s", name)
		}
	}
	for _, args := range [][]string{{"seamless", "upload"}, {"seamless", "upload", "--platform", "APPLE"}, {"seamless", "create"}, {"seamless", "create", "--body", "{"}} {
		command := NewRootCommand()
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)
		command.SetArgs(args)
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) = %v", args, err)
		}
	}
	command := NewRootCommand()
	command.SetArgs([]string{"seamless", "create", "--body", `{}`, "--alias", "invalid"})
	if err := command.Execute(); err == nil {
		t.Fatal("create accepted removed --alias scalar flag")
	}
	assertSeamlessBodyModes(t, seamlessFixtures(t)[1])
}

func executeSeamlessFixture(t *testing.T, row seamlessFixture, apiError bool) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != row.method || request.URL.Path != row.path || request.URL.RawQuery != "" {
			t.Errorf("request = %s %s?%s", request.Method, request.URL.Path, request.URL.RawQuery)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" || request.Header.Get("Accept") != "application/json" {
			t.Errorf("headers authorization=%q accept=%q", request.Header.Get("Authorization"), request.Header.Get("Accept"))
		}
		for _, name := range []string{"X-Esper-Tenant-ID", "X-Esper-Caller-ID", "X-Esper-User-ID", "X-Esper-Private-Service"} {
			if request.Header.Get(name) != "" {
				t.Errorf("internal header %s was sent", name)
			}
		}
		if row.multipart {
			assertSeamlessMultipart(t, request)
		} else {
			body, err := io.ReadAll(request.Body)
			if err != nil || string(body) != row.body || request.Header.Get("Content-Type") != "application/json" {
				t.Errorf("body=%q content-type=%q error=%v", body, request.Header.Get("Content-Type"), err)
			}
		}
		name, status := row.name+"-success.json", row.status
		if apiError {
			name, status = row.name+"-api-error.json", row.errorStatus
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(readSeamlessFixture(t, name))
	}))
	defer server.Close()
	command := configuredSeamlessCommand(t, server.URL)
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs(row.args)
	err := command.Execute()
	if apiError {
		var api *esperruntime.APIError
		if !errors.As(err, &api) || api.StatusCode != row.errorStatus || esperruntime.ExitCode(err) != 1 || output.Len() != 0 || stderr.Len() != 0 || !bytes.Equal(api.Body, readSeamlessFixture(t, row.name+"-api-error.json")) {
			t.Fatalf("API error=%v stdout=%q stderr=%q", err, output.Bytes(), stderr.Bytes())
		}
		return
	}
	if err != nil || stderr.Len() != 0 || !bytes.Equal(output.Bytes(), readSeamlessFixture(t, row.name+"-success.golden")) {
		t.Fatalf("error=%v stdout=%q stderr=%q", err, output.Bytes(), stderr.Bytes())
	}
}

func assertSeamlessMultipart(t *testing.T, request *http.Request) {
	t.Helper()
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("content type = %q, error = %v", request.Header.Get("Content-Type"), err)
	}
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	files := request.MultipartForm.File["csv_file"]
	if len(files) != 1 || request.FormValue("platform") != "WINDOWS" {
		t.Fatalf("files=%d platform=%q", len(files), request.FormValue("platform"))
	}
	file, err := files[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil || !bytes.Equal(data, []byte("serial_number\nSERIAL-UPLOAD-1\n")) {
		t.Fatalf("csv=%q error=%v", data, err)
	}
}

func assertSeamlessBodyModes(t *testing.T, row seamlessFixture) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(file, []byte(row.body), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{row.body, "@" + file, "-"} {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			body, _ := io.ReadAll(request.Body)
			if request.Method != row.method || request.URL.Path != row.path || string(body) != row.body || request.Header.Get("Content-Type") != "application/json" {
				t.Errorf("request=%s %s body=%q content-type=%q", request.Method, request.URL.Path, body, request.Header.Get("Content-Type"))
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{}`))
		}))
		command := configuredSeamlessCommand(t, server.URL)
		if value == "-" {
			command.SetIn(strings.NewReader(row.body))
		}
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)
		command.SetArgs([]string{"seamless", "create", "--body", value})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		server.Close()
	}
}

func configuredSeamlessCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func readSeamlessFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "seamless", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
