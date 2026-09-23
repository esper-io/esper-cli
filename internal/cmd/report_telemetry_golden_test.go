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
	reportEnterpriseID = "11111111-1111-4111-8111-111111111111"
	reportDeviceID     = "22222222-2222-4222-8222-222222222222"
	reportID           = "33333333-3333-4333-8333-333333333333"
	reportSubscription = "7"
	reportLocation     = "42"
	reportTile         = "24"
	reportBody         = `{"email_ids":["alerts@example.com"],"is_subscribed":true}`
	reportPatchBody    = `{"is_subscribed":false}`
	reportCreateBody   = `{"report_type":"device","start_time":"2026-08-01T00:00:00Z","end_time":"2026-08-02T00:00:00Z","is_download":true}`
)

type reportTelemetryFixture struct {
	id, document, name, method, path, body string
	args                                   []string
	query                                  url.Values
	status, errorStatus                    int
	pagination, destructive                bool
}

type reportTelemetryMetadata struct {
	Generation, Method, Path, Noun, Verb, Pagination, ScopeParent, SuccessMedia string
	Command                                                                     []string
	Destructive                                                                 bool
}

func reportTelemetryFixtures() []reportTelemetryFixture {
	return []reportTelemetryFixture{
		{"getEventFeed", "v1", "event-feed-list", http.MethodGet, "/v1/enterprise/" + reportEnterpriseID + "/device/" + reportDeviceID + "/report/eventfeed/", "", []string{"event-feed", "list", "--enterprise", reportEnterpriseID, "--device", reportDeviceID, "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"getDeviceLocations", "v1", "device-location-list", http.MethodGet, "/v1/enterprise/" + reportEnterpriseID + "/report/location/", "", []string{"device-location", "list", "--enterprise", reportEnterpriseID, "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"getSpecificLocations", "v1", "specific-location-get", http.MethodGet, "/v1/enterprise/" + reportEnterpriseID + "/report/location/" + reportLocation + "/", "", []string{"specific-location", "get", reportEnterpriseID, reportLocation, "--json"}, nil, 200, 404, false, false},
		{"getDeviceTileReports", "v1", "device-tile-report-list", http.MethodGet, "/v1/enterprise/" + reportEnterpriseID + "/report/device-tiles/", "", []string{"device-tile-report", "list", "--enterprise", reportEnterpriseID, "--seamless-info", "true", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"seamless_info": {"true"}, "limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"getDeviceTileReport", "v1", "device-tile-report-get", http.MethodGet, "/v1/enterprise/" + reportEnterpriseID + "/report/device-tiles/" + reportTile + "/", "", []string{"device-tile-report", "get", reportEnterpriseID, reportTile, "--json"}, nil, 200, 404, false, false},
		{"getStatusMetrics", "v1", "v1-status-metric-list", http.MethodGet, "/v1/enterprise/" + reportEnterpriseID + "/report/status-metrics/", "", []string{"api", "v1", "status-metric", "list", "--enterprise", reportEnterpriseID, "--json"}, nil, 200, 401, false, false},
		{"getStatusMetricsV2", "v2", "status-metric-list", http.MethodGet, "/v2/enterprise/" + reportEnterpriseID + "/report/status-metrics", "", []string{"status-metric", "list", "--enterprise", reportEnterpriseID, "--json"}, nil, 200, 401, false, false},
		{"getReportInfo", "legacy", "report-info-get", http.MethodGet, "/enterprise/report/info/", "", []string{"report-info", "get", "--json"}, nil, 200, 401, false, false},
		{"getSubscriptionReports", "legacy", "subscription-report-list", http.MethodGet, "/enterprise/report/subscription/", "", []string{"subscription-report", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"limit": {"1"}, "offset": {"0"}}, 200, 401, true, false},
		{"addSubscription", "legacy", "subscription-add", http.MethodPost, "/enterprise/report/subscription/", reportBody, []string{"subscription", "add", "--body", reportBody, "--json"}, nil, 201, 400, false, false},
		{"getSubscriptionReport", "legacy", "subscription-report-get", http.MethodGet, "/enterprise/report/subscription/" + reportSubscription + "/", "", []string{"subscription-report", "get", reportSubscription, "--json"}, nil, 200, 404, false, false},
		{"updateSubscriptionReport", "legacy", "subscription-report-update", http.MethodPut, "/enterprise/report/subscription/" + reportSubscription + "/", reportBody, []string{"subscription-report", "update", reportSubscription, "--body", reportBody, "--json"}, nil, 200, 400, false, false},
		{"patchSubscriptionReport", "legacy", "subscription-report-patch", http.MethodPatch, "/enterprise/report/subscription/" + reportSubscription + "/", "", []string{"subscription-report", "patch", reportSubscription, "--json"}, nil, 200, 400, false, false},
		{"deleteSubscriptionReport", "legacy", "subscription-report-delete", http.MethodDelete, "/enterprise/report/subscription/" + reportSubscription + "/", "", []string{"subscription-report", "delete", reportSubscription, "--yes"}, nil, 204, 404, false, true},
		{"getReportTypes", "v0", "report-type-list", http.MethodGet, "/report/v0/report-types", "", []string{"report-type", "list", "--json"}, nil, 200, 500, false, false},
		{"createReportStatus", "v0", "report-status-create", http.MethodPost, "/report/v0/reports", `{"end_time":"2026-08-02T00:00:00Z","is_download":true,"report_type":"device","start_time":"2026-08-01T00:00:00Z"}`, []string{"report-status", "create", "--report-type", "device", "--start-time", "2026-08-01T00:00:00Z", "--end-time", "2026-08-02T00:00:00Z", "--is-download", "true", "--json"}, nil, 200, 400, false, false},
		{"getReportStatus", "v0", "report-status-get", http.MethodGet, "/report/v0/reports/" + reportID, "", []string{"report-status", "get", reportID, "--json"}, nil, 200, 404, false, false},
	}
}

func TestReportTelemetryOperationCoverage(t *testing.T) {
	want := map[string]reportTelemetryMetadata{
		"getEventFeed":             {"v1", "GET", "/v1/enterprise/{enterprise_id}/device/{device_id}/report/eventfeed/", "event-feed", "list", "limit-offset", "device", "application/json", []string{"event-feed", "list"}, false},
		"getDeviceLocations":       {"v1", "GET", "/v1/enterprise/{enterprise_id}/report/location/", "device-location", "list", "limit-offset", "enterprise", "application/json", []string{"device-location", "list"}, false},
		"getSpecificLocations":     {"v1", "GET", "/v1/enterprise/{enterprise_id}/report/location/{location_id}/", "specific-location", "get", "none", "", "application/json", []string{"specific-location", "get"}, false},
		"getDeviceTileReports":     {"v1", "GET", "/v1/enterprise/{enterprise_id}/report/device-tiles/", "device-tile-report", "list", "limit-offset", "enterprise", "application/json", []string{"device-tile-report", "list"}, false},
		"getDeviceTileReport":      {"v1", "GET", "/v1/enterprise/{enterprise_id}/report/device-tiles/{device_tiles_id}/", "device-tile-report", "get", "none", "", "application/json", []string{"device-tile-report", "get"}, false},
		"getStatusMetrics":         {"v1", "GET", "/v1/enterprise/{enterprise_id}/report/status-metrics/", "status-metric", "list", "none", "enterprise", "application/json", []string{"api", "v1", "status-metric", "list"}, false},
		"getStatusMetricsV2":       {"v2", "GET", "/v2/enterprise/{enterprise_id}/report/status-metrics", "status-metric", "list", "none", "enterprise", "application/json", []string{"status-metric", "list"}, false},
		"getReportInfo":            {"legacy", "GET", "/enterprise/report/info/", "report-info", "get", "none", "", "application/json", []string{"report-info", "get"}, false},
		"getSubscriptionReports":   {"legacy", "GET", "/enterprise/report/subscription/", "subscription-report", "list", "limit-offset", "", "application/json", []string{"subscription-report", "list"}, false},
		"addSubscription":          {"legacy", "POST", "/enterprise/report/subscription/", "subscription", "add", "none", "", "application/json", []string{"subscription", "add"}, false},
		"getSubscriptionReport":    {"legacy", "GET", "/enterprise/report/subscription/{subscription_id}/", "subscription-report", "get", "none", "", "application/json", []string{"subscription-report", "get"}, false},
		"updateSubscriptionReport": {"legacy", "PUT", "/enterprise/report/subscription/{subscription_id}/", "subscription-report", "update", "none", "", "application/json", []string{"subscription-report", "update"}, false},
		"patchSubscriptionReport":  {"legacy", "PATCH", "/enterprise/report/subscription/{subscription_id}/", "subscription-report", "patch", "none", "", "application/json", []string{"subscription-report", "patch"}, false},
		"deleteSubscriptionReport": {"legacy", "DELETE", "/enterprise/report/subscription/{subscription_id}/", "subscription-report", "delete", "none", "", "", []string{"subscription-report", "delete"}, true},
		"getReportTypes":           {"v0", "GET", "/report/v0/report-types", "report-type", "list", "none", "", "application/json", []string{"report-type", "list"}, false},
		"createReportStatus":       {"v0", "POST", "/report/v0/reports", "report-status", "create", "none", "", "application/json", []string{"report-status", "create"}, false},
		"getReportStatus":          {"v0", "GET", "/report/v0/reports/{report_id}", "report-status", "get", "none", "", "application/json", []string{"report-status", "get"}, false},
	}
	got := map[string]reportTelemetryMetadata{}
	for _, operation := range generated.Operations() {
		if _, ok := want[operation.OperationID]; ok {
			got[operation.OperationID] = reportTelemetryMetadata{operation.Generation, operation.Method, operation.Path, operation.Noun, operation.Verb, operation.Pagination, operation.ScopeParent, operation.SuccessMedia, operation.Command, operation.Destructive}
		}
	}
	if len(want) != 17 || !reflect.DeepEqual(want, got) {
		t.Fatalf("report telemetry inventory = %#v, want %#v", got, want)
	}
}

func TestReportTelemetryFixtureInventory(t *testing.T) {
	want := map[string]bool{"README.md": true, "RERECORD_WITH_TENANT": true}
	for _, row := range reportTelemetryFixtures() {
		for _, suffix := range []string{"-success.json", "-success.golden", "-api-error.json"} {
			want[row.name+suffix] = true
		}
		if row.pagination {
			want[row.name+"-second-page.json"] = true
		}
	}
	entries, err := os.ReadDir(reportTelemetryFixtureDir())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, entry := range entries {
		got[entry.Name()] = true
	}
	if len(want) != 57 || !reflect.DeepEqual(want, got) {
		t.Fatalf("fixture inventory = %#v, want 57 exact files", got)
	}
}

func TestReportTelemetryFixturesMatchResponseContracts(t *testing.T) {
	for _, row := range reportTelemetryFixtures() {
		document := readReportTelemetryDocument(t, row.document)
		operation := fixtureSchemaOperation(t, document, operationByID(t, row.id).Path, strings.ToLower(row.method))
		if row.status == http.StatusNoContent {
			if len(readReportTelemetryFixture(t, row.name+"-success.json")) != 0 {
				t.Fatalf("%s 204 fixture is not empty", row.name)
			}
		} else {
			validateReportTelemetryFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json")
		}
		if row.pagination {
			validateReportTelemetryFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-second-page.json")
		}
		value := validateReportTelemetryFixture(t, document, fixtureSchemaResponseForStatus(document, operation, row.errorStatus), row.name+"-api-error.json")
		if row.document != "legacy" || row.id != "getTelemetryGraphData" {
			if object, ok := value.(map[string]any); !ok || object["status"] != float64(row.errorStatus) {
				t.Fatalf("%s error = %#v", row.name, value)
			}
		}
	}
}

func TestReportTelemetryGoldenFixtures(t *testing.T) {
	for _, row := range reportTelemetryFixtures() {
		t.Run(row.name, func(t *testing.T) { executeReportTelemetryFixture(t, row, false) })
	}
}
func TestReportTelemetryAPIErrors(t *testing.T) {
	for _, row := range reportTelemetryFixtures() {
		t.Run(row.name, func(t *testing.T) { executeReportTelemetryFixture(t, row, true) })
	}
}

func TestReportTelemetryPaginationMerges(t *testing.T) {
	for _, row := range reportTelemetryFixtures() {
		if !row.pagination {
			continue
		}
		t.Run(row.name, func(t *testing.T) {
			first, err := esperruntime.UnwrapLimitOffset(readReportTelemetryFixture(t, row.name+"-success.json"))
			if err != nil || first.Next == "" || first.Previous != "" {
				t.Fatalf("first page = %#v, %v", first, err)
			}
			second, err := esperruntime.UnwrapLimitOffset(readReportTelemetryFixture(t, row.name+"-second-page.json"))
			if err != nil || second.Next != "" || second.Previous == "" {
				t.Fatalf("second page = %#v, %v", second, err)
			}
			merged, err := esperruntime.MarshalMergedResults(append(first.Results, second.Results...))
			if err != nil {
				t.Fatal(err)
			}
			var value any
			if err := json.Unmarshal(merged, &value); err != nil {
				t.Fatal(err)
			}
			output, _ := json.MarshalIndent(value, "", "  ")
			output = append(output, '\n')
			if !bytes.Equal(output, readReportTelemetryFixture(t, row.name+"-success.golden")) {
				t.Fatalf("merged output = %s", output)
			}
		})
	}
}

func TestReportTelemetryInputRules(t *testing.T) {
	root := NewRootCommand()
	for _, path := range [][]string{{"event-feed", "list"}, {"device-location", "list"}, {"device-tile-report", "list"}, {"status-metric", "list"}, {"api", "v1", "status-metric", "list"}} {
		command, _, err := root.Find(path)
		if err != nil {
			t.Fatal(err)
		}
		if command.Flags().Lookup("enterprise") == nil {
			t.Fatalf("%s needs --enterprise", strings.Join(path, " "))
		}
	}
	event, _, _ := root.Find([]string{"event-feed", "list"})
	if event.Flags().Lookup("device") == nil {
		t.Fatal("event feed needs --device")
	}
	tile, _, _ := root.Find([]string{"device-tile-report", "list"})
	for _, name := range []string{"seamless-info", "limit", "offset"} {
		if tile.Flags().Lookup(name) == nil {
			t.Fatalf("tile list missing --%s", name)
		}
	}
	for _, name := range []string{"location", "has-seamless-info"} {
		if tile.Flags().Lookup(name) != nil {
			t.Fatalf("tile list exposes removed --%s", name)
		}
	}
	for _, path := range [][]string{{"subscription", "add"}, {"subscription-report", "update"}} {
		command, _, _ := root.Find(path)
		if command.Flags().Lookup("body") == nil || command.Flags().Lookup("email-ids") != nil {
			t.Fatalf("%s is not body-only", strings.Join(path, " "))
		}
	}
	patch, _, _ := root.Find([]string{"subscription-report", "patch"})
	for _, name := range []string{"body", "is-subscribed"} {
		if patch.Flags().Lookup(name) == nil {
			t.Fatalf("patch missing --%s", name)
		}
	}
	for _, path := range [][]string{{"report-type", "list"}, {"report-status", "create"}, {"report-status", "get"}} {
		command, _, _ := root.Find(path)
		for _, name := range []string{"x-esper-tenant-id", "x-esper-user-id"} {
			if command.Flags().Lookup(name) != nil {
				t.Fatalf("%s exposes --%s", strings.Join(path, " "), name)
			}
		}
	}
	for _, args := range [][]string{{"event-feed", "list", "--enterprise", reportEnterpriseID}, {"subscription", "add"}, {"subscription-report", "update", reportSubscription}, {"report-status", "create"}, {"report-status", "create", "--body", "{"}, {"report-status", "create", "--body", reportCreateBody, "--report-type", "device"}} {
		assertReportTelemetryUsage(t, args)
	}
	assertReportTelemetryRequest(t, []string{"subscription-report", "patch", reportSubscription}, "", reportTelemetryFixtures()[12])
	assertReportTelemetryRequest(t, []string{"subscription-report", "patch", reportSubscription, "--is-subscribed", "false"}, `{"is_subscribed":false}`, reportTelemetryFixture{method: http.MethodPatch, path: "/enterprise/report/subscription/" + reportSubscription + "/", status: 200})
	assertReportTelemetryRequest(t, []string{"subscription-report", "patch", reportSubscription, "--body", reportPatchBody}, reportPatchBody, reportTelemetryFixture{method: http.MethodPatch, path: "/enterprise/report/subscription/" + reportSubscription + "/", status: 200})
	for _, row := range []reportTelemetryFixture{reportTelemetryFixtures()[9], reportTelemetryFixtures()[11]} {
		assertReportTelemetryBodyModes(t, row)
	}
	assertReportTelemetryBodyModes(t, reportTelemetryFixture{args: []string{"report-status", "create"}, method: http.MethodPost, path: "/report/v0/reports", body: reportCreateBody, status: 200})
}

func TestReportTelemetryDeleteDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	command := configuredReportTelemetryCommand(t, server.URL)
	command.SetIn(strings.NewReader("no\n"))
	command.SetArgs([]string{"subscription-report", "delete", reportSubscription})
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("delete refusal = %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}

func executeReportTelemetryFixture(t *testing.T, row reportTelemetryFixture, apiError bool) {
	t.Helper()
	args := append([]string(nil), row.args...)
	if apiError && row.pagination {
		args = append(args[:len(args)-2], "--json")
	}
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != row.method || r.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, row.method, row.path)
		}
		if r.Header.Get("Authorization") != "Bearer fixture-key" || r.Header.Get("Accept") != "application/json" {
			t.Errorf("headers authorization=%q accept=%q", r.Header.Get("Authorization"), r.Header.Get("Accept"))
		}
		for _, name := range []string{"X-Esper-Tenant-ID", "X-Esper-User-ID"} {
			if value := r.Header.Get(name); value != "" {
				t.Errorf("internal header %s = %q", name, value)
			}
		}
		wantQuery := row.query
		if row.pagination && !apiError && requests == 2 {
			wantQuery = reportTelemetrySecondPageQuery(row)
		}
		if got := r.URL.Query(); got.Encode() != wantQuery.Encode() {
			t.Errorf("query = %q, want %q", got.Encode(), wantQuery.Encode())
		}
		assertReportTelemetryBody(t, r, row.body)
		name, status := row.name+"-success.json", row.status
		if apiError {
			name, status = row.name+"-api-error.json", row.errorStatus
		} else if row.pagination && requests == 2 {
			name = row.name + "-second-page.json"
		}
		response := readReportTelemetryFixture(t, name)
		if row.pagination && !apiError && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?"+reportTelemetrySecondPageQuery(row).Encode()), 1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(response)
	}))
	defer server.Close()
	command := configuredReportTelemetryCommand(t, server.URL)
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs(args)
	err := command.Execute()
	if apiError {
		var value *esperruntime.APIError
		if !errors.As(err, &value) || value.StatusCode != row.errorStatus || esperruntime.ExitCode(err) != 1 || output.Len() != 0 || stderr.Len() != 0 || !bytes.Equal(value.Body, readReportTelemetryFixture(t, row.name+"-api-error.json")) {
			t.Fatalf("API error = %v stdout=%q stderr=%q", err, output.Bytes(), stderr.Bytes())
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if row.pagination && requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
	if stderr.Len() != 0 || !bytes.Equal(output.Bytes(), readReportTelemetryFixture(t, row.name+"-success.golden")) {
		t.Fatalf("stdout=%q stderr=%q", output.Bytes(), stderr.Bytes())
	}
}

func reportTelemetrySecondPageQuery(row reportTelemetryFixture) url.Values {
	query := url.Values{}
	for key, values := range row.query {
		query[key] = append([]string(nil), values...)
	}
	query.Set("offset", "1")
	return query
}
func assertReportTelemetryBody(t *testing.T, request *http.Request, want string) {
	t.Helper()
	got, err := io.ReadAll(request.Body)
	if err != nil || string(got) != want {
		t.Errorf("body=%q err=%v want=%q", got, err, want)
	}
	if want != "" && request.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type=%q", request.Header.Get("Content-Type"))
	}
}
func assertReportTelemetryUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}
func assertReportTelemetryRequest(t *testing.T, args []string, body string, row reportTelemetryFixture) {
	assertReportTelemetryRequestWithInput(t, args, strings.NewReader(""), body, row)
}
func assertReportTelemetryRequestWithInput(t *testing.T, args []string, input io.Reader, body string, row reportTelemetryFixture) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != row.method || r.URL.Path != row.path {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		assertReportTelemetryBody(t, r, body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(row.status)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	command := configuredReportTelemetryCommand(t, server.URL)
	command.SetIn(input)
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
}
func assertReportTelemetryBodyModes(t *testing.T, row reportTelemetryFixture) {
	t.Helper()
	for _, mode := range []string{"inline", "file", "stdin"} {
		body, input := row.body, io.Reader(strings.NewReader(""))
		if mode == "file" {
			file := filepath.Join(t.TempDir(), "body.json")
			if err := os.WriteFile(file, []byte(row.body), 0o600); err != nil {
				t.Fatal(err)
			}
			body = "@" + file
		} else if mode == "stdin" {
			body, input = "-", strings.NewReader(row.body)
		}
		args := make([]string, 0, len(row.args)+2)
		for index := 0; index < len(row.args); index++ {
			if row.args[index] == "--body" {
				index++
				continue
			}
			if row.args[index] != "--json" {
				args = append(args, row.args[index])
			}
		}
		args = append(args, "--body", body)
		assertReportTelemetryRequestWithInput(t, args, input, row.body, row)
	}
}
func configuredReportTelemetryCommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}
func operationByID(t *testing.T, id string) generated.Operation {
	t.Helper()
	for _, operation := range generated.Operations() {
		if operation.OperationID == id {
			return operation
		}
	}
	t.Fatalf("operation %s not found", id)
	return generated.Operation{}
}
func reportTelemetryFixtureDir() string {
	return filepath.Join("..", "..", "spec", "fixtures", "report-telemetry")
}
func readReportTelemetryFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(reportTelemetryFixtureDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func readReportTelemetryDocument(t *testing.T, generation string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", generation+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}
func validateReportTelemetryFixture(t *testing.T, document, schema map[string]any, name string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal(readReportTelemetryFixture(t, name), &value); err != nil {
		t.Fatal(err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}
