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
)

type alarmAlertFixture struct {
	key, fixture, method, path, body string
	args                             []string
	query                            url.Values
	status, errorStatus              int
	all, destructive                 bool
}

func alarmAlertFixtures() []alarmAlertFixture {
	return []alarmAlertFixture{
		{"v1 GET /v1/enterprise/{enterprise_id}/alertchannels/", "alert-channel-list", "GET", "/v1/enterprise/enterprise-1/alertchannels/", "", []string{"alert-channel", "list", "--enterprise", "enterprise-1", "--name", "mail", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"name": {"mail"}, "limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"v1 POST /v1/enterprise/{enterprise_id}/alertchannels/", "alert-channel-create", "POST", "/v1/enterprise/enterprise-1/alertchannels/", `{"enterprise":"enterprise-1","name":"mail","type":"email","properties":{"to_address":["ops@example.test"]}}`, []string{"alert-channel", "create", "--enterprise", "enterprise-1", "--body", `{"name":"mail","type":"email","properties":{"to_address":["ops@example.test"]}}`, "--json"}, nil, 201, 400, false, false},
		{"v1 GET /v1/enterprise/{enterprise_id}/alertchannels/{alert_id}", "alert-channel-get", "GET", "/v1/enterprise/enterprise-1/alertchannels/1", "", []string{"alert-channel", "get", "1", "--enterprise", "enterprise-1", "--json"}, nil, 200, 401, false, false},
		{"v1 PUT /v1/enterprise/{enterprise_id}/alertchannels/{alert_id}", "alert-channel-update", "PUT", "/v1/enterprise/enterprise-1/alertchannels/1", `{"enterprise":"enterprise-1","name":"mail","type":"email","properties":{"to_address":["ops@example.test"]}}`, []string{"alert-channel", "update", "1", "--enterprise", "enterprise-1", "--body", `{"name":"mail","type":"email","properties":{"to_address":["ops@example.test"]}}`, "--json"}, nil, 200, 400, false, false},
		{"v1 PATCH /v1/enterprise/{enterprise_id}/alertchannels/{alert_id}", "alert-channel-patch", "PATCH", "/v1/enterprise/enterprise-1/alertchannels/1", `{"name":"mail-updated"}`, []string{"alert-channel", "patch", "1", "--enterprise", "enterprise-1", "--name", "mail-updated", "--json"}, nil, 200, 400, false, false},
		{"v1 DELETE /v1/enterprise/{enterprise_id}/alertchannels/{alert_id}", "alert-channel-delete", "DELETE", "/v1/enterprise/enterprise-1/alertchannels/1", "", []string{"alert-channel", "delete", "1", "--enterprise", "enterprise-1", "--yes", "--json"}, nil, 204, 400, false, true},
		{"v1 GET /v1/enterprise/{enterprise_id}/alarmrules/", "alarm-rule-list", "GET", "/v1/enterprise/enterprise-1/alarmrules/", "", []string{"alarm-rule", "list", "--enterprise", "enterprise-1", "--name", "cpu", "--is-active", "true", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"name": {"cpu"}, "is_active": {"true"}, "limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"v1 POST /v1/enterprise/{enterprise_id}/alarmrules/", "alarm-rule-add", "POST", "/v1/enterprise/enterprise-1/alarmrules/", `{"enterprise":"enterprise-1","name":"cpu","metric":"cpu","conditions":{"value":90,"unit":"Percent","comparator":"GreaterThanThreshold"},"realert_interval":5}`, []string{"alarm-rule", "add", "--enterprise", "enterprise-1", "--body", `{"name":"cpu","metric":"cpu","conditions":{"value":90,"unit":"Percent","comparator":"GreaterThanThreshold"},"realert_interval":5}`, "--json"}, nil, 201, 400, false, false},
		{"v1 GET /v1/enterprise/{enterprise_id}/alarmrules/{alarm_id}/", "alarm-rule-get", "GET", "/v1/enterprise/enterprise-1/alarmrules/alarm-1/", "", []string{"alarm-rule", "get", "alarm-1", "--enterprise", "enterprise-1", "--json"}, nil, 200, 401, false, false},
		{"v1 PUT /v1/enterprise/{enterprise_id}/alarmrules/{alarm_id}/", "alarm-rule-update", "PUT", "/v1/enterprise/enterprise-1/alarmrules/alarm-1/", `{"enterprise":"enterprise-1","name":"cpu","metric":"cpu","conditions":{"value":90,"unit":"Percent","comparator":"GreaterThanThreshold"},"realert_interval":5}`, []string{"alarm-rule", "update", "alarm-1", "--enterprise", "enterprise-1", "--body", `{"name":"cpu","metric":"cpu","conditions":{"value":90,"unit":"Percent","comparator":"GreaterThanThreshold"},"realert_interval":5}`, "--json"}, nil, 200, 400, false, false},
		{"v1 PATCH /v1/enterprise/{enterprise_id}/alarmrules/{alarm_id}/", "alarm-rule-patch", "PATCH", "/v1/enterprise/enterprise-1/alarmrules/alarm-1/", `{"name":"cpu-updated"}`, []string{"alarm-rule", "patch", "alarm-1", "--enterprise", "enterprise-1", "--name", "cpu-updated", "--json"}, nil, 200, 400, false, false},
		{"v1 DELETE /v1/enterprise/{enterprise_id}/alarmrules/{alarm_id}/", "alarm-rule-delete", "DELETE", "/v1/enterprise/enterprise-1/alarmrules/alarm-1/", "", []string{"alarm-rule", "delete", "alarm-1", "--enterprise", "enterprise-1", "--yes", "--json"}, nil, 204, 400, false, true},
		{"v1 GET /v1/enterprise/{enterprise_id}/alarmrules/{alarm_id}/alarmhistory/", "alarm-history-get", "GET", "/v1/enterprise/enterprise-1/alarmrules/alarm-1/alarmhistory/", "", []string{"alarm-history", "get", "--enterprise", "enterprise-1", "--alarm-rule", "alarm-1", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
	}
}

func TestAlarmAlertOperationCoverage(t *testing.T) {
	rows, actual := map[string]bool{}, map[string]bool{}
	for _, row := range alarmAlertFixtures() {
		rows[row.key] = true
	}
	if len(rows) != 13 {
		t.Fatalf("fixture rows = %d, want 13", len(rows))
	}
	for _, op := range generated.Operations() {
		if op.Noun == "alert-channel" || op.Noun == "alarm-rule" || op.Noun == "alarm-history" {
			actual[op.Generation+" "+op.Method+" "+op.Path] = true
		}
	}
	if !reflect.DeepEqual(rows, actual) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(rows), len(actual))
	}
}

func TestAlarmAlertCommandsGoldenFixtures(t *testing.T) {
	for _, row := range alarmAlertFixtures() {
		t.Run(row.fixture, func(t *testing.T) { executeAlarmAlertFixture(t, row, false) })
	}
}
func TestAlarmAlertCommandsAPIErrors(t *testing.T) {
	for _, row := range alarmAlertFixtures() {
		t.Run(row.fixture, func(t *testing.T) { executeAlarmAlertFixture(t, row, true) })
	}
}

func executeAlarmAlertFixture(t *testing.T, row alarmAlertFixture, apiError bool) {
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
		if row.all && requests == 2 {
			wantQuery := url.Values{"offset": {"1"}}
			if row.fixture == "alarm-history-get" {
				wantQuery = url.Values{"limit": {"50"}, "offset": {"50"}}
			}
			if got, want := r.URL.Query().Encode(), wantQuery.Encode(); got != want {
				t.Errorf("second page query = %q, want %q", got, want)
			}
			fixture = row.fixture + "-second-page.json"
		} else if r.URL.Query().Encode() != row.query.Encode() {
			t.Errorf("query = %q, want %q", r.URL.Query(), row.query)
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
		body := readAlarmAlertFixture(t, fixture)
		if row.all && requests == 1 {
			nextQuery := "?offset=1"
			if row.fixture == "alarm-history-get" {
				nextQuery = "?limit=50&offset=50"
			}
			body = bytes.Replace(body, []byte("NEXT_PAGE"), []byte(server.URL+row.path+nextQuery), 1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(row.args)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != status || esperruntime.ExitCode(err) != 1 {
			t.Fatalf("API error = %v", err)
		}
		if want := readAlarmAlertFixture(t, fixture); !bytes.Equal(value.Body, want) {
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
	if want := readAlarmAlertFixture(t, row.fixture+"-success.golden"); !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestAlarmAlertInputValidation(t *testing.T) {
	for _, args := range [][]string{{"alert-channel", "create", "--enterprise", "e"}, {"alert-channel", "update", "1", "--enterprise", "e"}, {"alarm-rule", "add", "--enterprise", "e"}, {"alarm-rule", "update", "a", "--enterprise", "e"}, {"alert-channel", "patch", "1", "--enterprise", "e", "--body", `{}`, "--name", "n"}, {"alarm-rule", "patch", "a", "--enterprise", "e", "--body", `{}`, "--name", "n"}, {"alert-channel", "get", "1"}, {"alarm-rule", "get", "a"}, {"alarm-history", "get", "--enterprise", "e"}} {
		c := NewRootCommand()
		c.SetArgs(args)
		if err := c.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) = %v", args, err)
		}
	}
	for _, args := range [][]string{{"alert-channel", "create", "--enterprise", "e", "--name", "mail"}, {"alarm-rule", "add", "--enterprise", "e", "--name", "cpu"}} {
		c := NewRootCommand()
		c.SetArgs(args)
		if err := c.Execute(); err == nil {
			t.Fatalf("unsafe scalar input accepted: %v", args)
		}
	}
}

func TestAlarmAlertDestructiveCommandsDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	for _, row := range alarmAlertFixtures() {
		if row.destructive {
			c := NewRootCommand()
			c.SetIn(strings.NewReader("no\n"))
			c.SetArgs(withoutYes(row.args))
			if err := c.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
				t.Fatalf("decline %s = %v", row.fixture, err)
			}
		}
	}
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}

func readAlarmAlertFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "alarm-alert", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
