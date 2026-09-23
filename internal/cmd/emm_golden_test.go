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

const (
	emmEnterpriseID = "tenant-1"
	emmDetailBody   = `{"callback_url":"https://example.test/callback","completion_token":"completion-1","enterprise_token":"enterprise-token-1","google_enterprise_id":"google-enterprise-1","signup_url":"https://example.test/signup","state":1}`
	emmAccountBody  = `{"data":"private-data-1","google_enterprise_id":"google-enterprise-1","is_active":true,"is_set":true,"key_id":"key-1","public_data":"public-data-1","type":1}`
)

type emmFixture struct {
	key, name, method, path, body string
	args                          []string
	query                         url.Values
	status, errorStatus           int
	pagination                    string
}

func emmFixtures() []emmFixture {
	return []emmFixture{
		{"v0 GET /v0/enterprise/{enterprise_id}/emm", "emm-list", http.MethodGet, "/v0/enterprise/" + emmEnterpriseID + "/emm", "", []string{"emm", "list", "--enterprise", emmEnterpriseID, "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusBadRequest, "limit-offset"},
		{"v0 GET /v0/enterprise/{enterprise_id}/emm/{emm_id}", "emm-instance-get", http.MethodGet, "/v0/enterprise/" + emmEnterpriseID + "/emm/17", "", []string{"emm-instance", "get", emmEnterpriseID, "17", "--json"}, nil, http.StatusOK, http.StatusUnauthorized, ""},
		{"v2 POST /v2/emm/enrollment/begin/", "emm-enrollment-begin-create", http.MethodPost, "/v2/emm/enrollment/begin/", `{"callback_url":"https://example.test/callback"}`, []string{"emm-enrollment-begin", "create", "--callback-url", "https://example.test/callback", "--json"}, nil, http.StatusCreated, http.StatusBadRequest, ""},
		{"v2 GET /v2/emm/details/", "emm-detail-list", http.MethodGet, "/v2/emm/details/", "", []string{"emm-detail", "list", "--all", "--json"}, nil, http.StatusOK, http.StatusBadRequest, "apps-envelope"},
		{"v2 POST /v2/emm/details/", "emm-detail-create", http.MethodPost, "/v2/emm/details/", emmDetailBody, []string{"emm-detail", "create", "--callback-url", "https://example.test/callback", "--completion-token", "completion-1", "--enterprise-token", "enterprise-token-1", "--google-enterprise-id", "google-enterprise-1", "--signup-url", "https://example.test/signup", "--state", "1", "--json"}, nil, http.StatusCreated, http.StatusBadRequest, ""},
		{"v2 POST /v2/emm/web-token/", "emm-web-token-create", http.MethodPost, "/v2/emm/web-token/", `{"google_enterprise_id":"google-enterprise-1","parent_url":"https://example.test/parent"}`, []string{"emm-web-token", "create", "--google-enterprise-id", "google-enterprise-1", "--parent-url", "https://example.test/parent", "--json"}, nil, http.StatusOK, http.StatusBadRequest, ""},
		{"v2 POST /v2/emm/enrollment/complete/", "emm-enrollment-complete-create", http.MethodPost, "/v2/emm/enrollment/complete/", `{"completion_token":"completion-1","enterprise_token":"enterprise-token-1"}`, []string{"emm-enrollment-complete", "create", "--completion-token", "completion-1", "--enterprise-token", "enterprise-token-1", "--json"}, nil, http.StatusCreated, http.StatusBadRequest, ""},
		{"v2 GET /v2/emm/accounts/", "emm-account-list", http.MethodGet, "/v2/emm/accounts/", "", []string{"emm-account", "list", "--all", "--json"}, nil, http.StatusOK, http.StatusBadRequest, "apps-envelope"},
		{"v2 POST /v2/emm/accounts/", "emm-account-create", http.MethodPost, "/v2/emm/accounts/", emmAccountBody, []string{"emm-account", "create", "--data", "private-data-1", "--google-enterprise-id", "google-enterprise-1", "--is-active", "true", "--is-set", "true", "--key-id", "key-1", "--public-data", "public-data-1", "--type", "1", "--json"}, nil, http.StatusCreated, http.StatusBadRequest, ""},
	}
}

func TestEMMOperationCoverage(t *testing.T) {
	want, got := map[string]bool{}, map[string]bool{}
	for _, row := range emmFixtures() {
		if want[row.key] {
			t.Fatalf("duplicate fixture row %s", row.key)
		}
		want[row.key] = true
	}
	if len(want) != 9 {
		t.Fatalf("fixture rows = %d, want 9", len(want))
	}
	for _, operation := range generated.Operations() {
		if strings.HasPrefix(operation.Noun, "emm") {
			got[operation.Generation+" "+operation.Method+" "+operation.Path] = true
			if operation.Destructive {
				t.Fatalf("%s is unexpectedly destructive", operation.Noun)
			}
			if operation.Noun == "emm-instance" && (len(operation.Parameters) != 2 || operation.Parameters[0].Name != "enterprise_id" || operation.Parameters[1].Name != "emm_id") {
				t.Fatalf("emm-instance positional order = %#v", operation.Parameters)
			}
		}
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(want), len(got))
	}
}

func TestEMMFixturesMatchResponseContracts(t *testing.T) {
	documents := map[string]map[string]any{"v0": readEMMDocument(t, "v0.yaml"), "v2": readEMMDocument(t, "v2.yaml")}
	for _, row := range emmFixtures() {
		parts := strings.SplitN(row.key, " ", 3)
		document := documents[parts[0]]
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		emmValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		if row.pagination != "" {
			emmValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-second-page.json")
		}
		value := emmValidateFixture(t, document, fixtureSchemaResponseForStatus(document, operation, row.errorStatus), row.name+"-api-error.json")
		field := "code"
		if parts[0] == "v0" {
			field = "status"
		}
		if object, ok := value.(map[string]any); !ok || object[field] != float64(row.errorStatus) {
			t.Fatalf("%s API error %s = %#v, want %d", row.name, field, value, row.errorStatus)
		}
	}
}

func TestEMMGoldenFixtures(t *testing.T) {
	for _, row := range emmFixtures() {
		t.Run(row.name, func(t *testing.T) { executeEMMFixture(t, row, false) })
	}
}

func TestEMMAPIErrors(t *testing.T) {
	for _, row := range emmFixtures() {
		t.Run(row.name, func(t *testing.T) { executeEMMFixture(t, row, true) })
	}
}

func TestEMMPaginationMerges(t *testing.T) {
	for _, row := range emmFixtures() {
		if row.pagination == "" {
			continue
		}
		t.Run(row.name, func(t *testing.T) {
			first, second := readEMMFixture(t, row.name+"-success.json"), readEMMFixture(t, row.name+"-second-page.json")
			var one, two esperruntime.Page
			var err error
			if row.pagination == "limit-offset" {
				one, err = esperruntime.UnwrapLimitOffset(first)
			} else {
				one, err = esperruntime.UnwrapAppsEnvelope(first)
			}
			if err != nil || one.Next == "" {
				t.Fatalf("first page = %#v, error = %v", one, err)
			}
			if row.pagination == "limit-offset" {
				two, err = esperruntime.UnwrapLimitOffset(second)
			} else {
				two, err = esperruntime.UnwrapAppsEnvelope(second)
			}
			if err != nil || two.Next != "" {
				t.Fatalf("second page = %#v, error = %v", two, err)
			}
			merged, err := esperruntime.MarshalMergedResults(append(one.Results, two.Results...))
			var value any
			var formatted []byte
			if err == nil {
				err = json.Unmarshal(merged, &value)
			}
			if err == nil {
				formatted, err = json.MarshalIndent(value, "", "  ")
				formatted = append(formatted, '\n')
			}
			if err != nil || !bytes.Equal(formatted, readEMMFixture(t, row.name+"-success.golden")) {
				t.Fatalf("merged = %s, error = %v", merged, err)
			}
		})
	}
}

func TestEMMInputRules(t *testing.T) {
	root := NewRootCommand()
	for _, row := range emmFixtures() {
		command, _, err := root.Find(row.args[:2])
		if err != nil {
			t.Fatal(err)
		}
		if command.Flags().Lookup("authorization") != nil {
			t.Fatalf("%s exposes internal authorization flag", strings.Join(row.args[:2], " "))
		}
	}

	assertEMMUsage(t, []string{"emm", "list"})

	for _, row := range emmFixtures() {
		if row.body == "" {
			continue
		}
		command, _, err := root.Find(row.args[:2])
		if err != nil || command.Flags().Lookup("body") == nil {
			t.Fatalf("%s body flag = %v, error = %v", row.name, command.Flags().Lookup("body"), err)
		}
		for _, flag := range requiredEMMFlags(row) {
			if command.Flags().Lookup(flag) == nil {
				t.Fatalf("%s missing required scalar flag --%s", row.name, flag)
			}
			assertEMMUsage(t, omitEMMFlag(row.args, flag))
		}
		assertEMMUsage(t, row.args[:2])
		assertEMMUsage(t, []string{row.args[0], row.args[1], "--body", "{"})
		mixed := append([]string{row.args[0], row.args[1], "--body", row.body}, row.args[2], row.args[3])
		assertEMMUsage(t, mixed)

		file := filepath.Join(t.TempDir(), row.name+".json")
		if err := os.WriteFile(file, []byte(row.body), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, body := range []string{row.body, "@" + file, "-"} {
			input := io.Reader(strings.NewReader(""))
			if body == "-" {
				input = strings.NewReader(row.body)
			}
			assertEMMRequest(t, []string{row.args[0], row.args[1], "--body", body}, input, row.method, row.path, row.body, row.name+"-success.json", row.status)
		}
	}
}

func omitEMMFlag(args []string, omitted string) []string {
	result := make([]string, 0, len(args)-2)
	for index := 0; index < len(args); index++ {
		if args[index] == "--"+omitted {
			index++
			continue
		}
		result = append(result, args[index])
	}
	return result
}

func requiredEMMFlags(row emmFixture) []string {
	switch row.name {
	case "emm-enrollment-begin-create":
		return []string{"callback-url"}
	case "emm-detail-create":
		return []string{"callback-url", "completion-token", "enterprise-token", "google-enterprise-id", "signup-url", "state"}
	case "emm-web-token-create":
		return []string{"google-enterprise-id", "parent-url"}
	case "emm-enrollment-complete-create":
		return []string{"completion-token", "enterprise-token"}
	case "emm-account-create":
		return []string{"data", "google-enterprise-id", "is-active", "is-set", "key-id", "public-data", "type"}
	default:
		return nil
	}
}

func executeEMMFixture(t *testing.T, row emmFixture, apiError bool) {
	t.Helper()
	arguments := append([]string(nil), row.args...)
	if apiError && row.pagination != "" {
		arguments = append(arguments[:len(arguments)-2], "--json")
	}
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != row.method || request.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, row.method, row.path)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" || request.Header.Get("Accept") != "application/json" {
			t.Errorf("authorization = %q, accept = %q", request.Header.Get("Authorization"), request.Header.Get("Accept"))
		}
		if row.pagination != "" && !apiError && requests == 2 {
			if got, want := request.URL.Query().Encode(), (url.Values{"offset": {"1"}}).Encode(); got != want {
				t.Errorf("second page query = %q, want %q", got, want)
			}
		} else if got, want := request.URL.Query().Encode(), row.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		assertEMMBody(t, request, row.body)
		name, status := row.name+"-success.json", row.status
		if apiError {
			name, status = row.name+"-api-error.json", row.errorStatus
		} else if row.pagination != "" && requests == 2 {
			name = row.name + "-second-page.json"
		}
		response := readEMMFixture(t, name)
		if row.pagination != "" && !apiError && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?offset=1"), 1)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(response)
	}))
	defer server.Close()
	command := configuredEMMCommand(t, server.URL)
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs(arguments)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != row.errorStatus || esperruntime.ExitCode(err) != 1 || output.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("API error = %v, stdout = %q, stderr = %q", err, output.String(), stderr.String())
		}
		if want := readEMMFixture(t, row.name+"-api-error.json"); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %q, want %q", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if row.pagination != "" && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if stderr.Len() != 0 || !bytes.Equal(output.Bytes(), readEMMFixture(t, row.name+"-success.golden")) {
		t.Fatalf("stdout = %q, stderr = %q", output.Bytes(), stderr.Bytes())
	}
}

func assertEMMRequest(t *testing.T, args []string, input io.Reader, method, path, body, fixture string, status int) {
	t.Helper()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != method || request.URL.Path != path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, method, path)
		}
		assertEMMBody(t, request, body)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(readEMMFixture(t, fixture))
	}))
	defer server.Close()
	command := configuredEMMCommand(t, server.URL)
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

func assertEMMBody(t *testing.T, request *http.Request, want string) {
	t.Helper()
	got, err := io.ReadAll(request.Body)
	if err != nil || string(got) != want {
		t.Errorf("body = %q, error = %v, want %q", got, err, want)
	}
	if want != "" && request.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type = %q, want application/json", request.Header.Get("Content-Type"))
	}
}

func assertEMMUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}

func configuredEMMCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func readEMMDocument(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", name))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func emmValidateFixture(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readEMMFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

func readEMMFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "emm", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
