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

	"github.com/esper-io/esper-cli/internal/cmd/generated"
	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

type tokenUserFixture struct {
	key, fixture, method, path, body string
	args                             []string
	query                            url.Values
	status, errorStatus              int
	all, destructive, raw, multipart bool
}

func tokenUserFixtures() []tokenUserFixture {
	return []tokenUserFixture{
		{"authn2 GET /authn2/v1/users/", "authn-user-list", "GET", "/authn2/v1/users/", "", []string{"authn-user", "list", "--legacy-user-id", "legacy-1", "--json"}, url.Values{"legacy_user_id": {"legacy-1"}}, 200, 400, false, false, false, false},
		{"v0 POST /tenant/v0/deptokens/", "dep-token-create", "POST", "/tenant/v0/deptokens/", "", []string{"dep-token", "create"}, nil, 200, 400, false, false, true, false},
		{"v0 GET /tenant/v0/deptokens/", "dep-token-list", "GET", "/tenant/v0/deptokens/", "", []string{"dep-token", "list", "--state", "CSR_GENERATED", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"state": {"CSR_GENERATED"}, "limit": {"1"}, "offset": {"0"}}, 200, 401, true, false, false, false},
		{"v0 GET /tenant/v0/deptokens/{id}/", "dep-token-based-on-get", "GET", "/tenant/v0/deptokens/token-1/", "", []string{"dep-token-based-on", "get", "token-1", "--json"}, nil, 200, 401, false, false, false, false},
		{"v0 PUT /tenant/v0/deptokens/{id}/", "dep-token-upload-update", "PUT", "/tenant/v0/deptokens/token-1/", "", []string{"dep-token-upload", "update", "token-1", "--file", "FIXTURE_FILE", "--token-name", "DEP token", "--json"}, nil, 200, 400, false, false, false, true},
		{"authz2 PUT /authz2/v1/users/{user_id}", "different-user-update", "PUT", "/authz2/v1/users/user-1", `{"groups":["group-1"]}`, []string{"different-user", "update", "user-1", "--body", `{"groups":["group-1"]}`, "--json"}, nil, 200, 400, false, false, false, false},
		{"authn2 POST /authn2/v0/tenant/{enterprise_id}/invite", "invite-create", "POST", "/authn2/v0/tenant/tenant-1/invite", `{"meta":{}}`, []string{"invite", "create", "--tenant", "tenant-1", "--body", `{"meta":{}}`, "--json"}, nil, 201, 400, false, false, false, false},
		{"authn2 GET /authn2/v0/tenant/{enterprise_id}/invite", "invite-list", "GET", "/authn2/v0/tenant/tenant-1/invite", "", []string{"invite", "list", "--tenant", "tenant-1", "--json"}, nil, 201, 400, false, false, false, false},
		{"authn2 PUT /authn2/v0/user/{user_id}", "own-user-update", "PUT", "/authn2/v0/user/user-1", `{"first_name":"Ada"}`, []string{"own-user", "update", "user-1", "--first-name", "Ada", "--json"}, nil, 200, 400, false, false, false, false},
		{"authn2 POST /authn2/v0/personal-access-token/", "personal-access-token-create", "POST", "/authn2/v0/personal-access-token/", `{"description":"fixture","expiry_at":1,"name":"fixture"}`, []string{"personal-access-token", "create", "--name", "fixture", "--description", "fixture", "--expiry-at", "1", "--json"}, nil, 200, 400, false, false, false, false},
		{"authn2 GET /authn2/v0/personal-access-token/", "personal-access-token-list", "GET", "/authn2/v0/personal-access-token/", "", []string{"personal-access-token", "list", "--json"}, nil, 200, 400, false, false, false, false},
		{"authn2 PUT /authn2/v0/personal-access-token/{personal_access_token_id}", "personal-access-token-update", "PUT", "/authn2/v0/personal-access-token/token-1", `{"expiry_at":1}`, []string{"personal-access-token", "update", "token-1", "--expiry-at", "1", "--json"}, nil, 200, 400, false, false, false, false},
		{"authn2 DELETE /authn2/v0/personal-access-token/{personal_access_token_id}", "personal-access-token-delete", "DELETE", "/authn2/v0/personal-access-token/token-1", "", []string{"personal-access-token", "delete", "token-1", "--yes", "--json"}, nil, 204, 400, false, true, false, false},
		{"v0 POST /v0/enterprise/{enterprise_id}/developerapp/{developerapp_id}/renew-token/", "renew-token-renew", "POST", "/v0/enterprise/enterprise-1/developerapp/app-1/renew-token/", "", []string{"renew-token", "renew", "enterprise-1", "app-1", "--access-token", "access-1", "--json"}, url.Values{"access_token": {"access-1"}}, 201, 400, false, false, false, false},
		{"authn2 DELETE /authn2/v0/tenant/{enterprise_id}/user/{user_id}/", "tenant-user-delete", "DELETE", "/authn2/v0/tenant/tenant-1/user/user-1/", "", []string{"tenant-user", "delete", "user-1", "--tenant", "tenant-1", "--yes", "--json"}, nil, 204, 400, false, true, false, false},
		{"authn2 DELETE /authn2/v0/tenant/{enterprise_id}/invite/{invite_id}", "tenant-user-invite-delete", "DELETE", "/authn2/v0/tenant/tenant-1/invite/invite-1", "", []string{"tenant-user-invite", "delete", "invite-1", "--tenant", "tenant-1", "--yes", "--json"}, nil, 204, 400, false, true, false, false},
		{"v0 GET /tenant/v0/vpptokens/", "tenant-vpptoken-list", "GET", "/tenant/v0/vpptokens/", "", []string{"tenant-vpptoken", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false, false, false},
		{"v0 POST /tenant/v0/vpptokens/", "tenant-vpptoken-create", "POST", "/tenant/v0/vpptokens/", "", []string{"tenant-vpptoken", "create", "--filename", "FIXTURE_FILE", "--json"}, nil, 201, 400, false, false, false, true},
		{"v0 DELETE /tenant/v0/vpptokens/{id}", "tenant-vpptoken-delete", "DELETE", "/tenant/v0/vpptokens/1", "", []string{"tenant-vpptoken", "delete", "1", "--yes", "--json"}, nil, 204, 401, false, true, false, false},
		{"v1 GET /v1/token-info/", "token-info-get", "GET", "/v1/token-info/", "", []string{"token-info", "get", "--json"}, nil, 200, 401, false, false, false, false},
		{"legacy GET /user/", "user-list", "GET", "/user/", "", []string{"user", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false, false, false},
		{"legacy POST /user/", "user-create", "POST", "/user/", `{"profile":{}}`, []string{"user", "create", "--body", `{"profile":{}}`, "--json"}, nil, 201, 400, false, false, false, false},
		{"legacy GET /user/{user_id}/", "user-get", "GET", "/user/1/", "", []string{"user", "get", "1", "--json"}, nil, 200, 401, false, false, false, false},
		{"legacy PUT /user/{user_id}/", "user-update", "PUT", "/user/1/", `{"profile":{}}`, []string{"user", "update", "1", "--body", `{"profile":{}}`, "--json"}, nil, 200, 400, false, false, false, false},
		{"legacy PATCH /user/{user_id}/", "user-partial-update", "PATCH", "/user/1/", `{"first_name":"Ada"}`, []string{"user", "partial-update", "1", "--first-name", "Ada", "--json"}, nil, 200, 400, false, false, false, false},
		{"legacy DELETE /user/{user_id}/", "user-delete-delete", "DELETE", "/user/1/", "", []string{"user-delete", "delete", "1", "--yes", "--json"}, nil, 204, 401, false, true, false, false},
		{"legacy GET /user_info/", "user-info-get", "GET", "/user_info/", "", []string{"user-info", "get", "--json"}, nil, 200, 401, false, false, false, false},
		{"v0 GET /v0/enterprise/{enterprise_id}/emm/{emm_id}/webtoken/", "webtoken-list", "GET", "/v0/enterprise/enterprise-1/emm/1/webtoken/", "", []string{"webtoken", "list", "--enterprise", "enterprise-1", "--emm", "1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false, false, false},
		{"v0 POST /v0/enterprise/{enterprise_id}/emm/{emm_id}/webtoken/", "webtoken-create", "POST", "/v0/enterprise/enterprise-1/emm/1/webtoken/", `{"emm":1,"parent_url":"https://example.test"}`, []string{"webtoken", "create", "--enterprise", "enterprise-1", "--emm", "1", "--parent-url", "https://example.test", "--json"}, nil, 201, 401, false, false, false, false},
		{"v0 GET /v0/enterprise/{enterprise_id}/emm/{emm_id}/webtoken/{webtoken_id}", "webtoken-instance-get", "GET", "/v0/enterprise/enterprise-1/emm/1/webtoken/token-1", "", []string{"webtoken-instance", "get", "token-1", "--enterprise", "enterprise-1", "--emm", "1", "--json"}, nil, 200, 401, false, false, false, false},
		{"v0 PUT /v0/enterprise/{enterprise_id}/emm/{emm_id}/webtoken/{webtoken_id}", "webtoken-instance-update", "PUT", "/v0/enterprise/enterprise-1/emm/1/webtoken/token-1", `{"emm":1,"parent_url":"https://example.test"}`, []string{"webtoken-instance", "update", "token-1", "--enterprise", "enterprise-1", "--emm", "1", "--parent-url", "https://example.test", "--json"}, nil, 200, 400, false, false, false, false},
		{"v0 PATCH /v0/enterprise/{enterprise_id}/emm/{emm_id}/webtoken/{webtoken_id}", "webtoken-instance-patch", "PATCH", "/v0/enterprise/enterprise-1/emm/1/webtoken/token-1", `{"token":"updated"}`, []string{"webtoken-instance", "patch", "token-1", "--enterprise", "enterprise-1", "--emm", "1", "--token", "updated", "--json"}, nil, 200, 400, false, false, false, false},
		{"v0 DELETE /v0/enterprise/{enterprise_id}/emm/{emm_id}/webtoken/{webtoken_id}", "webtoken-instance-delete", "DELETE", "/v0/enterprise/enterprise-1/emm/1/webtoken/token-1", "", []string{"webtoken-instance", "delete", "token-1", "--enterprise", "enterprise-1", "--emm", "1", "--yes", "--json"}, nil, 204, 400, false, true, false, false},
	}
}

func TestTokenUserOperationCoverage(t *testing.T) {
	nouns := map[string]bool{"authn-user": true, "dep-token": true, "dep-token-based-on": true, "dep-token-upload": true, "different-user": true, "invite": true, "own-user": true, "personal-access-token": true, "renew-token": true, "tenant-user": true, "tenant-user-invite": true, "tenant-vpptoken": true, "token-info": true, "user": true, "user-delete": true, "user-info": true, "webtoken": true, "webtoken-instance": true}
	rows := map[string]bool{}
	for _, row := range tokenUserFixtures() {
		rows[row.key] = true
	}
	if len(rows) != 33 {
		t.Fatalf("fixture rows = %d, want 33", len(rows))
	}
	actual := map[string]bool{}
	for _, op := range generated.Operations() {
		if nouns[op.Noun] {
			actual[op.Generation+" "+op.Method+" "+op.Path] = true
		}
	}
	if !reflect.DeepEqual(rows, actual) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(rows), len(actual))
	}
}

func TestTokenUserCommandsGoldenFixtures(t *testing.T) {
	for _, row := range tokenUserFixtures() {
		t.Run(row.fixture, func(t *testing.T) { executeTokenUserFixture(t, row, false) })
	}
}
func TestTokenUserCommandsAPIErrors(t *testing.T) {
	for _, row := range tokenUserFixtures() {
		t.Run(row.fixture, func(t *testing.T) { executeTokenUserFixture(t, row, true) })
	}
}

func executeTokenUserFixture(t *testing.T, row tokenUserFixture, apiError bool) {
	t.Helper()
	fixture, status := row.fixture+"-success.json", row.status
	if apiError {
		fixture, status = row.fixture+"-api-error.json", row.errorStatus
	}
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != row.method || r.URL.Path != row.path {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if row.raw && r.Header.Get("Accept") != "text/plain; charset=utf-8" {
			t.Errorf("accept = %q, want text/plain; charset=utf-8", r.Header.Get("Accept"))
		}
		if row.all && requests == 2 {
			if got, want := r.URL.Query().Encode(), (url.Values{"offset": {"1"}}).Encode(); got != want {
				t.Errorf("second page query = %q, want %q", got, want)
			}
			fixture = row.fixture + "-second-page.json"
		} else if r.URL.Query().Encode() != row.query.Encode() {
			t.Errorf("query = %q, want %q", r.URL.Query(), row.query)
		}
		if row.multipart {
			validateTokenUserMultipart(t, r, row.fixture)
		}
		if row.body != "" {
			got, _ := io.ReadAll(r.Body)
			var a, b any
			_ = json.Unmarshal(got, &a)
			_ = json.Unmarshal([]byte(row.body), &b)
			if !reflect.DeepEqual(a, b) {
				t.Errorf("body = %s, want %s", got, row.body)
			}
		}
		body := readTokenUserFixture(t, fixture)
		if row.all && requests == 1 {
			body = bytes.Replace(body, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?offset=1"), 1)
		}
		if !row.raw {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	args := append([]string(nil), row.args...)
	for i, value := range args {
		if value == "FIXTURE_FILE" {
			path := filepath.Join(t.TempDir(), "token.p7m")
			if err := os.WriteFile(path, []byte("token fixture"), 0o600); err != nil {
				t.Fatal(err)
			}
			args[i] = path
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
		if want := readTokenUserFixture(t, fixture); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %s, want %s", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if row.all && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	want := readTokenUserFixture(t, row.fixture+"-success.golden")
	if !bytes.Equal(bytes.TrimSpace(output.Bytes()), bytes.TrimSpace(want)) {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestTokenUserInputValidation(t *testing.T) {
	for _, args := range [][]string{{"dep-token", "create", "--json"}, {"dep-token-upload", "update", "id", "--file", "x"}, {"different-user", "update", "id"}, {"invite", "create", "--tenant", "t"}, {"own-user", "update", "id"}, {"personal-access-token", "create", "--name", "n"}, {"user", "create"}, {"user", "partial-update", "1"}, {"user", "partial-update", "1", "--body", `{}`, "--first-name", "Ada"}, {"webtoken", "create", "--enterprise", "e", "--emm", "1"}, {"webtoken-instance", "get", "id", "--enterprise", "e"}} {
		c := NewRootCommand()
		c.SetArgs(args)
		if err := c.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) = %v", args, err)
		}
	}
}

func TestTokenUserRawAndOptionalMultipartBehavior(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path == "/tenant/v0/vpptokens/" && r.ContentLength != 0 {
			t.Errorf("optional VPP request content length = %d, want 0", r.ContentLength)
		}
		if r.URL.Path == "/tenant/v0/deptokens/" {
			_, _ = w.Write([]byte("DEP token fixture\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(readTokenUserFixture(t, "tenant-vpptoken-create-success.json"))
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")

	command := NewRootCommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs([]string{"dep-token", "create"})
	if err := command.Execute(); err != nil || stdout.String() != "DEP token fixture\n" {
		t.Fatalf("raw stdout = %q, error = %v", stdout.String(), err)
	}

	output := filepath.Join(t.TempDir(), "dep-token.txt")
	command = NewRootCommand()
	command.SetArgs([]string{"dep-token", "create", "--output", output})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(output); err != nil || string(data) != "DEP token fixture\n" {
		t.Fatalf("raw output file = %q, error = %v", data, err)
	}

	command = NewRootCommand()
	command.SetArgs([]string{"tenant-vpptoken", "create", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
}

func TestTokenUserDestructiveCommandsDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	for _, row := range tokenUserFixtures() {
		if !row.destructive {
			continue
		}
		c := NewRootCommand()
		c.SetIn(strings.NewReader("no\n"))
		c.SetArgs(withoutYes(row.args))
		if err := c.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("decline %s = %v", row.fixture, err)
		}
	}
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}

func readTokenUserFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "token-user", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func validateTokenUserMultipart(t *testing.T, request *http.Request, fixture string) {
	t.Helper()
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("content type = %q, want multipart/form-data", request.Header.Get("Content-Type"))
	}
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	name := "file"
	if fixture == "tenant-vpptoken-create" {
		name = "filename"
	} else if values := request.MultipartForm.Value["token_name"]; len(values) != 1 || values[0] != "DEP token" {
		t.Fatalf("token_name = %q, want DEP token", values)
	}
	files := request.MultipartForm.File[name]
	if len(files) != 1 {
		t.Fatalf("multipart file %s = %d, want 1", name, len(files))
	}
	file, err := files[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil || string(data) != "token fixture" {
		t.Fatalf("multipart file %s = %q, error = %v", name, data, err)
	}
}
