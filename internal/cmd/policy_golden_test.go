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
	policyEnterpriseID = "11111111-1111-4111-8111-111111111111"
	policyID           = "42"
	policyCreateInput  = `{"enterprise":"https://wrong.invalid/api/enterprise/wrong/","name":"Kiosk baseline","description":null,"policy":{"kioskMode":true,"applications":[]},"is_active":true}`
	policyCreateBody   = `{"description":null,"enterprise":"API_BASE/enterprise/11111111-1111-4111-8111-111111111111/","is_active":true,"name":"Kiosk baseline","policy":{"applications":[],"kioskMode":true}}`
	policyUpdateBody   = `{"enterprise":"API_BASE/enterprise/11111111-1111-4111-8111-111111111111/?format=json","name":"Kiosk baseline","description":null,"policy":{"applications":[],"kioskMode":true},"is_active":true}`
	policyPatchBody    = `{"is_active":false,"name":"Kiosk baseline updated"}`
)

type policyFixture struct {
	key, name, method, path, body string
	args                          []string
	query                         url.Values
	status, errorStatus           int
	all                           bool
}

type policyOperationMetadata struct {
	Generation, Method, Path, Noun, Verb, Pagination, ScopeParent, Summary, SuccessMedia string
	Command                                                                              []string
}

func policyFixtures() []policyFixture {
	return []policyFixture{
		{"legacy GET /enterprise/{enterprise_id}/policy/", "legacy-policy-list", http.MethodGet, "/enterprise/" + policyEnterpriseID + "/policy/", "", []string{"policy", "list", "--enterprise", policyEnterpriseID, "--name", "Kiosk", "--is-active", "true", "--created-on-gt", "2026-08-01T00:00:00Z", "--created-on-lt", "2026-08-31T23:59:59Z", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"name": {"Kiosk"}, "is_active": {"true"}, "created_on_gt": {"2026-08-01T00:00:00Z"}, "created_on_lt": {"2026-08-31T23:59:59Z"}, "limit": {"1"}, "offset": {"0"}}, http.StatusOK, http.StatusUnauthorized, true},
		{"legacy POST /enterprise/{enterprise_id}/policy/", "legacy-policy-create", http.MethodPost, "/enterprise/" + policyEnterpriseID + "/policy/", policyCreateBody, []string{"policy", "create", "--enterprise", policyEnterpriseID, "--body", policyCreateInput, "--json"}, nil, http.StatusCreated, http.StatusBadRequest, false},
		{"legacy GET /enterprise/{enterprise_id}/policy/{policy_id}/", "legacy-policy-get", http.MethodGet, "/enterprise/" + policyEnterpriseID + "/policy/" + policyID + "/", "", []string{"policy", "get", policyEnterpriseID, policyID, "--json"}, nil, http.StatusOK, http.StatusNotFound, false},
		{"legacy PUT /enterprise/{enterprise_id}/policy/{policy_id}/", "legacy-policy-update", http.MethodPut, "/enterprise/" + policyEnterpriseID + "/policy/" + policyID + "/", policyUpdateBody, []string{"policy", "update", policyEnterpriseID, policyID, "--body", policyUpdateBody, "--json"}, nil, http.StatusOK, http.StatusBadRequest, false},
		{"legacy PATCH /enterprise/{enterprise_id}/policy/{policy_id}/", "legacy-policy-partial-update", http.MethodPatch, "/enterprise/" + policyEnterpriseID + "/policy/" + policyID + "/", policyPatchBody, []string{"policy", "partial-update", policyEnterpriseID, policyID, "--name", "Kiosk baseline updated", "--is-active", "false", "--json"}, nil, http.StatusOK, http.StatusBadRequest, false},
	}
}

func TestPolicyOperationCoverage(t *testing.T) {
	want := map[string]policyOperationMetadata{
		"listPolicies":        {"legacy", http.MethodGet, "/enterprise/{enterprise_id}/policy/", "policy", "list", "limit-offset", "enterprise", "List all policies in enterprise", "application/json", []string{"policy", "list"}},
		"createPolicy":        {"legacy", http.MethodPost, "/enterprise/{enterprise_id}/policy/", "policy", "create", "none", "enterprise", "Create a new Enterprise Policy", "application/json", []string{"policy", "create"}},
		"getPolicyById":       {"legacy", http.MethodGet, "/enterprise/{enterprise_id}/policy/{policy_id}/", "policy", "get", "none", "", "Get Enterprise Policy", "application/json", []string{"policy", "get"}},
		"updatePolicy":        {"legacy", http.MethodPut, "/enterprise/{enterprise_id}/policy/{policy_id}/", "policy", "update", "none", "", "Update Enterprise Policy", "application/json", []string{"policy", "update"}},
		"partialupdatePolicy": {"legacy", http.MethodPatch, "/enterprise/{enterprise_id}/policy/{policy_id}/", "policy", "partial-update", "none", "", "Partial update EnterprisePolicy", "application/json", []string{"policy", "partial-update"}},
	}
	got := map[string]policyOperationMetadata{}
	for _, operation := range generated.Operations() {
		if operation.Noun == "policy" {
			got[operation.OperationID] = policyOperationMetadata{operation.Generation, operation.Method, operation.Path, operation.Noun, operation.Verb, operation.Pagination, operation.ScopeParent, operation.Summary, operation.SuccessMedia, operation.Command}
		}
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("generated policy inventory = %#v, want %#v", got, want)
	}
	var urlAutoFillOperations []string
	for _, operation := range generated.Operations() {
		if operation.Body != nil {
			for _, fill := range operation.Body.AutoFill {
				if fill.Format == "url" || fill.Format == "uri" {
					urlAutoFillOperations = append(urlAutoFillOperations, operation.OperationID)
				}
			}
		}
		switch operation.OperationID {
		case "createPolicy":
			if operation.Body == nil || len(operation.Body.AutoFill) != 1 || operation.Body.AutoFill[0].Name != "enterprise" || operation.Body.AutoFill[0].Format != "url" {
				t.Fatalf("create policy auto-fill = %#v", operation.Body)
			}
		case "updatePolicy":
			if operation.Body == nil || len(operation.Body.AutoFill) != 0 {
				t.Fatalf("update policy auto-fill = %#v", operation.Body)
			}
		}
	}
	if !reflect.DeepEqual(urlAutoFillOperations, []string{"createPolicy"}) {
		t.Fatalf("URL auto-fill operations = %v, want only createPolicy", urlAutoFillOperations)
	}
}

func TestPolicyFixtureInventory(t *testing.T) {
	want := map[string]bool{"README.md": true, "RERECORD_WITH_TENANT": true}
	for _, row := range policyFixtures() {
		want[row.name+"-success.json"] = true
		want[row.name+"-success.golden"] = true
		want[row.name+"-api-error.json"] = true
		if row.all {
			want[row.name+"-second-page.json"] = true
		}
	}
	entries, err := os.ReadDir(filepath.Join("..", "..", "spec", "fixtures", "policy"))
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]bool, len(entries))
	for _, entry := range entries {
		got[entry.Name()] = true
	}
	if len(want) != 18 || !reflect.DeepEqual(want, got) {
		t.Fatalf("policy fixture inventory = %#v, want 18 exact files", got)
	}
}

func TestPolicyFixturesMatchResponseContracts(t *testing.T) {
	document := readPolicyDocument(t)
	for _, row := range policyFixtures() {
		parts := strings.SplitN(row.key, " ", 3)
		operation := fixtureSchemaOperation(t, document, parts[2], strings.ToLower(parts[1]))
		policyValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		if row.all {
			policyValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-second-page.json")
		}
		value := policyValidateFixture(t, document, fixtureSchemaResponseForStatus(document, operation, row.errorStatus), row.name+"-api-error.json")
		if object, ok := value.(map[string]any); !ok || object["status"] != float64(row.errorStatus) {
			t.Fatalf("%s API error = %#v, want status %d", row.name, value, row.errorStatus)
		}
	}
}

func TestPolicyGoldenFixtures(t *testing.T) {
	for _, row := range policyFixtures() {
		t.Run(row.name, func(t *testing.T) { executePolicyFixture(t, row, false) })
	}
}

func TestPolicyAPIErrors(t *testing.T) {
	for _, row := range policyFixtures() {
		t.Run(row.name, func(t *testing.T) { executePolicyFixture(t, row, true) })
	}
}

func TestPolicyPaginationMerges(t *testing.T) {
	first, second := readPolicyFixture(t, "legacy-policy-list-success.json"), readPolicyFixture(t, "legacy-policy-list-second-page.json")
	one, err := esperruntime.UnwrapLimitOffset(first)
	if err != nil || one.Next == "" || one.Previous != "" {
		t.Fatalf("first page = %#v, error = %v", one, err)
	}
	two, err := esperruntime.UnwrapLimitOffset(second)
	if err != nil || two.Next != "" || two.Previous == "" {
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
	if err != nil || !bytes.Equal(formatted, readPolicyFixture(t, "legacy-policy-list-success.golden")) {
		t.Fatalf("merged = %s, error = %v", merged, err)
	}
}

func TestPolicyInputRules(t *testing.T) {
	root := NewRootCommand()
	for _, path := range [][]string{{"policy", "list"}, {"policy", "create"}} {
		command, _, err := root.Find(path)
		if err != nil || command.Flags().Lookup("enterprise") == nil {
			t.Fatalf("%s enterprise scope flag = %v, error = %v", strings.Join(path, " "), command.Flags().Lookup("enterprise"), err)
		}
	}
	create, _, _ := root.Find([]string{"policy", "create"})
	update, _, _ := root.Find([]string{"policy", "update"})
	for _, command := range []*cobra.Command{create, update} {
		if command.Flags().Lookup("body") == nil {
			t.Fatalf("%s body flag missing", command.CommandPath())
		}
		for _, name := range []string{"enterprise", "name", "description", "google-policy-id", "policy", "is-template", "is-active"} {
			if command.Flags().Lookup(name) != nil && name != "enterprise" {
				t.Fatalf("%s unexpectedly exposes --%s", command.CommandPath(), name)
			}
		}
	}
	partial, _, _ := root.Find([]string{"policy", "partial-update"})
	for _, name := range []string{"enterprise", "name", "description", "google-policy-id", "is-template", "is-active", "body"} {
		if partial.Flags().Lookup(name) == nil {
			t.Fatalf("policy partial-update --%s missing", name)
		}
	}
	if partial.Flags().Lookup("policy") != nil {
		t.Fatal("policy partial-update unexpectedly exposes complex --policy")
	}
	for _, args := range [][]string{
		{"policy", "list"},
		{"policy", "create", "--enterprise", policyEnterpriseID},
		{"policy", "create", "--enterprise", policyEnterpriseID, "--body", "{"},
		{"policy", "update", policyEnterpriseID, policyID},
		{"policy", "update", policyEnterpriseID, policyID, "--body", "{"},
		{"policy", "partial-update", policyEnterpriseID, policyID},
		{"policy", "partial-update", policyEnterpriseID, policyID, "--body", "{"},
		{"policy", "partial-update", policyEnterpriseID, policyID, "--body", policyPatchBody, "--name", "conflict"},
	} {
		assertPolicyUsage(t, args)
	}

	for _, row := range []policyFixture{policyFixtures()[1], policyFixtures()[3], policyFixtures()[4]} {
		body := policyPatchBody
		if row.name == "legacy-policy-create" {
			body = policyCreateInput
		} else if row.name == "legacy-policy-update" {
			body = strings.ReplaceAll(policyUpdateBody, "API_BASE", "ENDPOINT")
		}
		for _, mode := range []string{"inline", "file", "stdin"} {
			assertPolicyBodyMode(t, row, body, mode)
		}
	}
}

func executePolicyFixture(t *testing.T, row policyFixture, apiError bool) {
	t.Helper()
	requests := 0
	fixture, status := row.name+"-success.json", row.status
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != row.method || request.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, row.method, row.path)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" || request.Header.Get("Accept") != "application/json" {
			t.Errorf("authorization = %q, accept = %q", request.Header.Get("Authorization"), request.Header.Get("Accept"))
		}
		if row.all && !apiError && requests == 2 {
			if got, want := request.URL.Query().Encode(), policySecondPageQuery(row).Encode(); got != want {
				t.Errorf("second page query = %q, want %q", got, want)
			}
			fixture = row.name + "-second-page.json"
		} else if got, want := request.URL.Query().Encode(), row.query.Encode(); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		assertPolicyBody(t, request, strings.ReplaceAll(row.body, "API_BASE", server.URL))
		response := readPolicyFixture(t, fixture)
		if row.all && !apiError && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?"+policySecondPageQuery(row).Encode()), 1)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(response)
	}))
	defer server.Close()

	arguments := replacePolicyBase(row.args, server.URL)
	if apiError {
		fixture, status = row.name+"-api-error.json", row.errorStatus
		if row.all {
			arguments = append(arguments[:len(arguments)-2], "--json")
		}
	}
	command := configuredPolicyCommand(t, server.URL)
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs(arguments)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != status || esperruntime.ExitCode(err) != 1 || output.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("API error = %v, stdout = %q, stderr = %q", err, output.String(), stderr.String())
		}
		if want := readPolicyFixture(t, fixture); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %q, want %q", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if row.all && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if stderr.Len() != 0 || !bytes.Equal(output.Bytes(), readPolicyFixture(t, row.name+"-success.golden")) {
		t.Fatalf("stdout = %q, stderr = %q", output.Bytes(), stderr.Bytes())
	}
}

func assertPolicyBodyMode(t *testing.T, row policyFixture, body, mode string) {
	t.Helper()
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		assertPolicyBody(t, request, strings.ReplaceAll(row.body, "API_BASE", server.URL))
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(row.status)
		_, _ = writer.Write(readPolicyFixture(t, row.name+"-success.json"))
	}))
	defer server.Close()
	arguments := []string{"policy", strings.TrimPrefix(row.name, "legacy-policy-")}
	if row.name == "legacy-policy-create" {
		arguments = append(arguments, "--enterprise", policyEnterpriseID)
	} else {
		arguments = append(arguments, policyID, policyEnterpriseID)
	}
	resolvedBody := strings.ReplaceAll(body, "ENDPOINT", server.URL)
	inputBody := resolvedBody
	input := io.Reader(strings.NewReader(""))
	switch mode {
	case "file":
		file := filepath.Join(t.TempDir(), row.name+".json")
		if err := os.WriteFile(file, []byte(resolvedBody), 0o600); err != nil {
			t.Fatal(err)
		}
		inputBody = "@" + file
	case "stdin":
		inputBody = "-"
		input = strings.NewReader(resolvedBody)
	}
	arguments = append(arguments, "--body", inputBody)
	command := configuredPolicyCommand(t, server.URL)
	command.SetIn(input)
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute(%v): %v", arguments, err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func policySecondPageQuery(row policyFixture) url.Values {
	query := make(url.Values, len(row.query))
	for name, values := range row.query {
		query[name] = append([]string(nil), values...)
	}
	query.Set("offset", "1")
	return query
}

func assertPolicyBody(t *testing.T, request *http.Request, want string) {
	t.Helper()
	got, err := io.ReadAll(request.Body)
	if err != nil || string(got) != want {
		t.Errorf("body = %q, error = %v, want %q", got, err, want)
	}
	if want != "" && request.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type = %q, want application/json", request.Header.Get("Content-Type"))
	}
}

func assertPolicyUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}

func replacePolicyBase(values []string, base string) []string {
	result := append([]string(nil), values...)
	for index := range result {
		result[index] = strings.ReplaceAll(result[index], "API_BASE", base)
	}
	return result
}

func configuredPolicyCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func readPolicyDocument(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "legacy.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func policyValidateFixture(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	data := readPolicyFixture(t, name)
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s is not JSON: %v", name, err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

func readPolicyFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "policy", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
