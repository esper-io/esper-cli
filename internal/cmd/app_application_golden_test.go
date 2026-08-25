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

type appApplicationFixtureTest struct {
	key, name           string
	arguments           []string
	method, path        string
	query               url.Values
	body, fixture       string
	status, errorStatus int
	all, destructive    bool
	multipart           map[string]string
}

func appApplicationFixtureTests() []appApplicationFixtureTest {
	return []appApplicationFixtureTest{
		{"legacy DELETE /enterprise/{enterprise_id}/application/{application_id}/", "api legacy application delete", []string{"api", "legacy", "application", "delete", "fixture-id", "fixture-id", "--yes", "--json"}, "DELETE", "/enterprise/fixture-id/application/fixture-id/", nil, "", "legacy-delete--enterprise-enterprise-id-application-application-id", 204, 401, false, true, nil},
		{"legacy DELETE /enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/", "api legacy app-version delete", []string{"api", "legacy", "app-version", "delete", "fixture-id", "fixture-id", "fixture-id", "--yes", "--json"}, "DELETE", "/enterprise/fixture-id/application/fixture-id/version/fixture-id/", nil, "", "legacy-delete--enterprise-enterprise-id-application-application-id-version-version-id", 200, 404, false, true, nil},
		{"legacy GET /enterprise/{enterprise_id}/application/", "api legacy application list", []string{"api", "legacy", "application", "list", "--enterprise", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/enterprise/fixture-id/application/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "legacy-get--enterprise-enterprise-id-application", 200, 401, true, false, nil},
		{"legacy GET /enterprise/{enterprise_id}/application/{application_id}/", "api legacy application get", []string{"api", "legacy", "application", "get", "fixture-id", "fixture-id", "--json"}, "GET", "/enterprise/fixture-id/application/fixture-id/", nil, "", "legacy-get--enterprise-enterprise-id-application-application-id", 200, 401, false, false, nil},
		{"legacy GET /enterprise/{enterprise_id}/application/{application_id}/version/", "api legacy version list", []string{"api", "legacy", "version", "list", "--application", "fixture-id", "--enterprise", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/enterprise/fixture-id/application/fixture-id/version/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "legacy-get--enterprise-enterprise-id-application-application-id-version", 200, 401, true, false, nil},
		{"legacy GET /enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/", "api legacy app-version get", []string{"api", "legacy", "app-version", "get", "fixture-id", "fixture-id", "fixture-id", "--json"}, "GET", "/enterprise/fixture-id/application/fixture-id/version/fixture-id/", nil, "", "legacy-get--enterprise-enterprise-id-application-application-id-version-version-id", 200, 401, false, false, nil},
		{"legacy GET /enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/installdevices", "api legacy installdevice list", []string{"api", "legacy", "installdevice", "list", "--version", "fixture-id", "--application", "fixture-id", "--enterprise", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/enterprise/fixture-id/application/fixture-id/version/fixture-id/installdevices", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "legacy-get--enterprise-enterprise-id-application-application-id-version-version-id-installdevices", 200, 401, true, false, nil},
		{"legacy GET /enterprise/{enterprise_id}/device/{device_id}/app/", "api legacy app list", []string{"api", "legacy", "app", "list", "--enterprise", "fixture-id", "--device", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/enterprise/fixture-id/device/fixture-id/app/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "legacy-get--enterprise-enterprise-id-device-device-id-app", 200, 401, true, false, nil},
		{"legacy GET /enterprise/{enterprise_id}/device/{device_id}/app/{app_id}/", "device-app get", []string{"device-app", "get", "fixture-id", "fixture-id", "fixture-id", "--json"}, "GET", "/enterprise/fixture-id/device/fixture-id/app/fixture-id/", nil, "", "legacy-get--enterprise-enterprise-id-device-device-id-app-app-id", 200, 401, false, false, nil},
		{"legacy GET /enterprise/{enterprise_id}/device/{device_id}/install/", "api legacy install list", []string{"api", "legacy", "install", "list", "--enterprise", "fixture-id", "--device", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/enterprise/fixture-id/device/fixture-id/install/", url.Values{"device": {"fixture-id"}, "limit": {"1"}, "offset": {"0"}}, "", "legacy-get--enterprise-enterprise-id-device-device-id-install", 200, 401, true, false, nil},
		{"legacy PATCH /enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/", "api legacy app-version patch", []string{"api", "legacy", "app-version", "patch", "fixture-id", "fixture-id", "fixture-id", "--approval-status", "fixture-value", "--json"}, "PATCH", "/enterprise/fixture-id/application/fixture-id/version/fixture-id/", nil, "{\"approval_status\":\"fixture-value\"}", "legacy-patch--enterprise-enterprise-id-application-application-id-version-version-id", 200, 400, false, false, nil},
		{"legacy POST /enterprise/{enterprise_id}/application/upload/", "application upload", []string{"application", "upload", "fixture-id", "--app-file", "FIXTURE_FILE", "--json"}, "POST", "/enterprise/fixture-id/application/upload/", nil, "", "legacy-post--enterprise-enterprise-id-application-upload", 201, 415, false, false, map[string]string{"app_file": "fixture upload"}},
		{"v0 DELETE /v0/enterprise/{enterprise_id}/emm/{emm_id}/product/{product_id}", "app-instance delete", []string{"app-instance", "delete", "fixture-id", "fixture-id", "fixture-id", "--yes", "--json"}, "DELETE", "/v0/enterprise/fixture-id/emm/fixture-id/product/fixture-id", nil, "", "v0-delete--v0-enterprise-enterprise-id-emm-emm-id-product-product-id", 204, 400, false, true, nil},
		{"v0 GET /apps/v0/vpp", "app-vpp list", []string{"app-vpp", "list", "--json"}, "GET", "/apps/v0/vpp", nil, "", "v0-get--apps-v0-vpp", 200, 401, false, false, nil},
		{"v0 GET /apps/v0/vpp/{appId}", "app-vpp get", []string{"app-vpp", "get", "fixture-id", "--json"}, "GET", "/apps/v0/vpp/fixture-id", nil, "", "v0-get--apps-v0-vpp-appid", 200, 401, false, false, nil},
		{"v0 GET /v0/enterprise/{enterprise_id}/emm/{emm_id}/product/", "product list", []string{"product", "list", "--enterprise", "fixture-id", "--emm", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v0/enterprise/fixture-id/emm/fixture-id/product/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v0-get--v0-enterprise-enterprise-id-emm-emm-id-product", 200, 401, true, false, nil},
		{"v0 GET /v0/enterprise/{enterprise_id}/emm/{emm_id}/product/{product_id}", "app-info get", []string{"app-info", "get", "fixture-id", "fixture-id", "fixture-id", "--json"}, "GET", "/v0/enterprise/fixture-id/emm/fixture-id/product/fixture-id", nil, "", "v0-get--v0-enterprise-enterprise-id-emm-emm-id-product-product-id", 200, 401, false, false, nil},
		{"v0 GET /v0/enterprise/{enterprise_id}/emm/{emm_id}/product/{product_id}/install/", "install list", []string{"install", "list", "--enterprise", "fixture-id", "--emm", "fixture-id", "--product", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v0/enterprise/fixture-id/emm/fixture-id/product/fixture-id/install/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v0-get--v0-enterprise-enterprise-id-emm-emm-id-product-product-id-install", 200, 401, true, false, nil},
		{"v0 PATCH /v0/enterprise/{enterprise_id}/emm/{emm_id}/product/{product_id}", "app-info patch", []string{"app-info", "patch", "fixture-id", "fixture-id", "fixture-id", "--icon-url", "fixture-value", "--json"}, "PATCH", "/v0/enterprise/fixture-id/emm/fixture-id/product/fixture-id", nil, "{\"icon_url\":\"fixture-value\"}", "v0-patch--v0-enterprise-enterprise-id-emm-emm-id-product-product-id", 200, 400, false, false, nil},
		{"v0 POST /v0/enterprise/{enterprise_id}/emm/{emm_id}/product/", "product add", []string{"product", "add", "--enterprise", "fixture-id", "--emm", "fixture-id", "--created-on", "fixture-value", "--json"}, "POST", "/v0/enterprise/fixture-id/emm/fixture-id/product/", nil, "{\"created_on\":\"fixture-value\"}", "v0-post--v0-enterprise-enterprise-id-emm-emm-id-product", 201, 400, false, false, nil},
		{"v0 PUT /v0/enterprise/{enterprise_id}/emm/{emm_id}/product/{product_id}", "app-info update", []string{"app-info", "update", "fixture-id", "fixture-id", "fixture-id", "--icon-url", "fixture-value", "--json"}, "PUT", "/v0/enterprise/fixture-id/emm/fixture-id/product/fixture-id", nil, "{\"icon_url\":\"fixture-value\"}", "v0-put--v0-enterprise-enterprise-id-emm-emm-id-product-product-id", 200, 400, false, false, nil},
		{"v1 DELETE /v1/enterprise/{enterprise_id}/application/{application_id}/", "application delete", []string{"application", "delete", "fixture-id", "fixture-id", "--yes", "--json"}, "DELETE", "/v1/enterprise/fixture-id/application/fixture-id/", nil, "", "v1-delete--v1-enterprise-enterprise-id-application-application-id", 204, 401, false, true, nil},
		{"v1 DELETE /v1/enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/", "app-version delete", []string{"app-version", "delete", "fixture-id", "fixture-id", "fixture-id", "--yes", "--json"}, "DELETE", "/v1/enterprise/fixture-id/application/fixture-id/version/fixture-id/", nil, "", "v1-delete--v1-enterprise-enterprise-id-application-application-id-version-version-id", 200, 400, false, true, nil},
		{"v1 GET /v1/enterprise/{enterprise_id}/application/", "application list", []string{"application", "list", "--enterprise", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v1/enterprise/fixture-id/application/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v1-get--v1-enterprise-enterprise-id-application", 200, 401, true, false, nil},
		{"v1 GET /v1/enterprise/{enterprise_id}/application/{application_id}/", "application get", []string{"application", "get", "fixture-id", "fixture-id", "--json"}, "GET", "/v1/enterprise/fixture-id/application/fixture-id/", nil, "", "v1-get--v1-enterprise-enterprise-id-application-application-id", 200, 401, false, false, nil},
		{"v1 GET /v1/enterprise/{enterprise_id}/application/{application_id}/version/", "api v1 version list", []string{"api", "v1", "version", "list", "--application", "fixture-id", "--enterprise", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v1/enterprise/fixture-id/application/fixture-id/version/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v1-get--v1-enterprise-enterprise-id-application-application-id-version", 200, 401, true, false, nil},
		{"v1 GET /v1/enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/", "app-version get", []string{"app-version", "get", "fixture-id", "fixture-id", "fixture-id", "--json"}, "GET", "/v1/enterprise/fixture-id/application/fixture-id/version/fixture-id/", nil, "", "v1-get--v1-enterprise-enterprise-id-application-application-id-version-version-id", 200, 401, false, false, nil},
		{"v1 GET /v1/enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/installdevices", "installdevice list", []string{"installdevice", "list", "--version", "fixture-id", "--application", "fixture-id", "--enterprise", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v1/enterprise/fixture-id/application/fixture-id/version/fixture-id/installdevices", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v1-get--v1-enterprise-enterprise-id-application-application-id-version-version-id-installdevices", 200, 401, true, false, nil},
		{"v1 GET /v1/enterprise/{enterprise_id}/applications-minimal/", "application-minimal get", []string{"application-minimal", "get", "fixture-id", "--json"}, "GET", "/v1/enterprise/fixture-id/applications-minimal/", nil, "", "v1-get--v1-enterprise-enterprise-id-applications-minimal", 200, 404, false, false, nil},
		{"v1 PATCH /v1/enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/", "app-version patch", []string{"app-version", "patch", "fixture-id", "fixture-id", "fixture-id", "--approval-status", "fixture-value", "--json"}, "PATCH", "/v1/enterprise/fixture-id/application/fixture-id/version/fixture-id/", nil, "{\"approval_status\":\"fixture-value\"}", "v1-patch--v1-enterprise-enterprise-id-application-application-id-version-version-id", 200, 401, false, false, nil},
		{"v2 DELETE /v2/tenant-apps/{appId}/versions/{versionId}", "tenant-app-version delete", []string{"tenant-app-version", "delete", "fixture-id", "fixture-id", "--yes", "--json"}, "DELETE", "/v2/tenant-apps/fixture-id/versions/fixture-id", nil, "", "v2-delete--v2-tenant-apps-appid-versions-versionid", 200, 400, false, true, nil},
		{"v2 DELETE /v2/webclips/{webclipId}", "webclip delete", []string{"webclip", "delete", "fixture-id", "--yes", "--json"}, "DELETE", "/v2/webclips/fixture-id", nil, "", "v2-delete--v2-webclips-webclipid", 200, 409, false, true, nil},
		{"v2 GET /v2/appleappstore/", "appleappstore get", []string{"appleappstore", "get", "--title", "fixture-value", "--json"}, "GET", "/v2/appleappstore/", url.Values{"title": {"fixture-value"}}, "", "v2-get--v2-appleappstore", 200, 400, false, false, nil},
		{"v2 GET /v2/apps", "app list", []string{"app", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v2/apps", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v2-get--v2-apps", 200, 400, true, false, nil},
		{"v2 GET /v2/blueprints/{blueprint_id}/versions/", "version list", []string{"version", "list", "--blueprint", "fixture-id", "--json"}, "GET", "/v2/blueprints/fixture-id/versions/", nil, "", "v2-get--v2-blueprints-blueprint-id-versions", 200, 400, false, false, nil},
		{"v2 GET /v2/device-apps", "device-app list", []string{"device-app", "list", "--app-id", "fixture-value", "--app-version-id", "fixture-value", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v2/device-apps", url.Values{"app_id": {"fixture-value"}, "app_version_id": {"fixture-value"}, "limit": {"1"}, "offset": {"0"}}, "", "v2-get--v2-device-apps", 200, 400, true, false, nil},
		{"v2 GET /v2/devices/{deviceId}/device-apps/", "device-app list", []string{"device-app", "list", "--device", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v2/devices/fixture-id/device-apps/", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v2-get--v2-devices-deviceid-device-apps", 200, 400, true, false, nil},
		{"v2 GET /v2/esper-apps/{appId}/versions/{versionId}", "esper-app-version get", []string{"esper-app-version", "get", "fixture-id", "fixture-id", "--json"}, "GET", "/v2/esper-apps/fixture-id/versions/fixture-id", nil, "", "v2-get--v2-esper-apps-appid-versions-versionid", 200, 400, false, false, nil},
		{"v2 GET /v2/itunesapps", "itunesapp list", []string{"itunesapp", "list", "--app-id", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v2/itunesapps", url.Values{"app_id": {"fixture-id"}, "limit": {"1"}, "offset": {"0"}}, "", "v2-get--v2-itunesapps", 200, 400, true, false, nil},
		{"v2 GET /v2/preferred-regions", "preferred-region list", []string{"preferred-region", "list", "--json"}, "GET", "/v2/preferred-regions", nil, "", "v2-get--v2-preferred-regions", 200, 401, false, false, nil},
		{"v2 GET /v2/provisioning-profiles/{id}/versions", "version list", []string{"version", "list", "--provisioning-profile", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v2/provisioning-profiles/fixture-id/versions", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v2-get--v2-provisioning-profiles-id-versions", 200, 400, true, false, nil},
		{"v2 GET /v2/tenant-apps", "tenant-app list", []string{"tenant-app", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v2/tenant-apps", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v2-get--v2-tenant-apps", 200, 400, true, false, nil},
		{"v2 GET /v2/tenant-apps/{appId}", "tenant-app get", []string{"tenant-app", "get", "fixture-id", "--json"}, "GET", "/v2/tenant-apps/fixture-id", nil, "", "v2-get--v2-tenant-apps-appid", 200, 400, false, false, nil},
		{"v2 GET /v2/tenant-apps/{appId}/versions", "version list", []string{"version", "list", "--tenant-app", "fixture-id", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v2/tenant-apps/fixture-id/versions", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v2-get--v2-tenant-apps-appid-versions", 200, 400, true, false, nil},
		{"v2 GET /v2/tenant-apps/{appId}/versions/{versionId}", "tenant-app-version get", []string{"tenant-app-version", "get", "fixture-id", "fixture-id", "--json"}, "GET", "/v2/tenant-apps/fixture-id/versions/fixture-id", nil, "", "v2-get--v2-tenant-apps-appid-versions-versionid", 200, 400, false, false, nil},
		{"v2 GET /v2/tenant-esper-apps", "tenant-esper-app list", []string{"tenant-esper-app", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v2/tenant-esper-apps", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v2-get--v2-tenant-esper-apps", 200, 401, true, false, nil},
		{"v2 GET /v2/webclips", "webclip list", []string{"webclip", "list", "--limit", "1", "--offset", "0", "--all", "--json"}, "GET", "/v2/webclips", url.Values{"limit": {"1"}, "offset": {"0"}}, "", "v2-get--v2-webclips", 200, 400, true, false, nil},
		{"v2 GET /v2/webclips/{webclipId}", "webclip get", []string{"webclip", "get", "fixture-id", "--json"}, "GET", "/v2/webclips/fixture-id", nil, "", "v2-get--v2-webclips-webclipid", 200, 400, false, false, nil},
		{"v2 POST /v2/preferred-regions", "preferred-region create", []string{"preferred-region", "create", "--body", "{\"fixture\":\"packet-3\"}", "--json"}, "POST", "/v2/preferred-regions", nil, "{\"fixture\":\"packet-3\"}", "v2-post--v2-preferred-regions", 200, 400, false, false, nil},
		{"v2 POST /v2/tenant-apps", "tenant-app create", []string{"tenant-app", "create", "--file", "FIXTURE_FILE", "--json"}, "POST", "/v2/tenant-apps", nil, "", "v2-post--v2-tenant-apps", 200, 401, false, false, map[string]string{"file": "fixture upload"}},
		{"v2 POST /v2/webclips", "webclip create", []string{"webclip", "create", "--esper-name", "fixture-value", "--is-full-screen", "true", "--is-removable", "false", "--label", "fixture-value", "--target-bundle-id", "fixture-value", "--target-bundle-name", "fixture-value", "--url", "fixture-value", "--json"}, "POST", "/v2/webclips", nil, "", "v2-post--v2-webclips", 200, 400, false, false, map[string]string{"esper_name": "fixture-value", "is_full_screen": "true", "is_removable": "false", "label": "fixture-value", "target_bundle_id": "fixture-value", "target_bundle_name": "fixture-value", "url": "fixture-value"}},
		{"v2 PUT /v2/itunesapps/{appleAppId}/preferred-region", "preferred-region update", []string{"preferred-region", "update", "fixture-id", "--preferred-region", "fixture-value", "--json"}, "PUT", "/v2/itunesapps/fixture-id/preferred-region", nil, "{\"preferred_region\":\"fixture-value\"}", "v2-put--v2-itunesapps-appleappid-preferred-region", 200, 401, false, false, nil},
		{"v2 PUT /v2/tenant-apps/{appId}/versions/{versionId}", "tenant-app-version update", []string{"tenant-app-version", "update", "fixture-id", "fixture-id", "--description", "fixture-value", "--json"}, "PUT", "/v2/tenant-apps/fixture-id/versions/fixture-id", nil, "{\"description\":\"fixture-value\"}", "v2-put--v2-tenant-apps-appid-versions-versionid", 200, 404, false, false, nil},
	}
}

func TestAppApplicationOperationCoverage(t *testing.T) {
	nouns := map[string]bool{"app": true, "app-info": true, "app-instance": true, "app-version": true, "app-vpp": true, "appleappstore": true, "application": true, "application-minimal": true, "device-app": true, "esper-app-version": true, "install": true, "installdevice": true, "itunesapp": true, "preferred-region": true, "product": true, "tenant-app": true, "tenant-app-version": true, "tenant-esper-app": true, "version": true, "webclip": true}
	expected := map[string]bool{}
	for _, test := range appApplicationFixtureTests() {
		if expected[test.key] {
			t.Fatalf("duplicate explicit fixture row %s", test.key)
		}
		expected[test.key] = true
	}
	if len(expected) != 53 {
		t.Fatalf("fixture rows = %d, want 53", len(expected))
	}
	actual := map[string]bool{}
	for _, operation := range generated.Operations() {
		if nouns[operation.Noun] {
			actual[operation.Generation+" "+operation.Method+" "+operation.Path] = true
		}
	}
	if len(actual) != 53 || !reflect.DeepEqual(expected, actual) {
		t.Fatalf("packet operation keys mismatch: rows=%d generated=%d", len(expected), len(actual))
	}
}

func TestAppApplicationCommandsGoldenFixtures(t *testing.T) {
	for _, test := range appApplicationFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executeAppApplicationFixture(t, test, false) })
	}
}
func TestAppApplicationCommandsAPIErrors(t *testing.T) {
	for _, test := range appApplicationFixtureTests() {
		t.Run(test.name, func(t *testing.T) { executeAppApplicationFixture(t, test, true) })
	}
}

func TestAppApplicationInputValidation(t *testing.T) {
	for _, arguments := range [][]string{{"preferred-region", "create"}, {"app-info", "patch", "fixture-id", "fixture-id", "fixture-id", "--body", "{}", "--icon-url", "https://example.test/icon.png"}, {"device-app", "list"}, {"itunesapp", "list"}} {
		command := NewRootCommand()
		command.SetArgs(arguments)
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) error = %v", arguments, err)
		}
	}
}

func TestAppApplicationDestructiveCommandsDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	for _, test := range appApplicationFixtureTests() {
		if !test.destructive {
			continue
		}
		args := withoutYes(test.arguments)
		command := NewRootCommand()
		command.SetIn(strings.NewReader("no\n"))
		command.SetArgs(args)
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) error = %v", args, err)
		}
	}
	if requests != 0 {
		t.Fatalf("declined destructive requests = %d, want 0", requests)
	}
}

func executeAppApplicationFixture(t *testing.T, test appApplicationFixtureTest, apiError bool) {
	t.Helper()
	requests := 0
	fixture := test.fixture + "-success.json"
	status := test.status
	if apiError {
		fixture = test.fixture + "-api-error.json"
		status = test.errorStatus
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != test.method || request.URL.Path != test.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, test.method, test.path)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		if test.all && requests == 2 {
			if request.URL.Query().Get("offset") != "1" {
				t.Errorf("second-page offset = %q, want 1", request.URL.Query().Get("offset"))
			}
			fixture = test.fixture + "-second-page.json"
		} else if request.URL.Query().Encode() != test.query.Encode() {
			t.Errorf("query = %q, want %q", request.URL.Query(), test.query)
		}
		assertAppApplicationBody(t, request, test)
		response := readAppApplicationFixture(t, fixture)
		if test.all && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+test.path+"?offset=1"), 1)
		}
		if len(response) > 0 {
			writer.Header().Set("Content-Type", "application/json")
		}
		writer.WriteHeader(status)
		_, _ = writer.Write(response)
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	args := append([]string(nil), test.arguments...)
	for i, arg := range args {
		if arg == "FIXTURE_FILE" {
			file := filepath.Join(t.TempDir(), "fixture.bin")
			if err := os.WriteFile(file, []byte("fixture upload"), 0o600); err != nil {
				t.Fatal(err)
			}
			args[i] = file
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
		if want := readAppApplicationFixture(t, fixture); !bytes.Equal(value.Body, want) {
			t.Fatalf("API error body = %s, want %s", value.Body, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if test.all && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	golden := readAppApplicationFixture(t, test.fixture+"-success.golden")
	if len(golden) == 0 {
		if output.Len() != 0 {
			t.Fatalf("raw output = %q, want empty", output.String())
		}
		return
	}
	var got, want any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(golden, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", golden, output.Bytes())
	}
}

func assertAppApplicationBody(t *testing.T, request *http.Request, test appApplicationFixtureTest) {
	t.Helper()
	if len(test.multipart) > 0 {
		mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Errorf("content type = %q", request.Header.Get("Content-Type"))
		}
		if err := request.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		for name, want := range test.multipart {
			if files := request.MultipartForm.File[name]; len(files) > 0 {
				file, err := files[0].Open()
				if err != nil {
					t.Fatal(err)
				}
				got, _ := io.ReadAll(file)
				file.Close()
				if string(got) != want {
					t.Errorf("multipart file %s = %q, want %q", name, got, want)
				}
			} else if got := request.MultipartForm.Value[name]; len(got) != 1 || got[0] != want {
				t.Errorf("multipart field %s = %q, want %q", name, got, want)
			}
		}
		return
	}
	data, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if test.body == "" {
		if len(data) != 0 {
			t.Errorf("body = %q, want empty", data)
		}
		return
	}
	if request.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type = %q", request.Header.Get("Content-Type"))
	}
	var got, want any
	if json.Unmarshal(data, &got) != nil || json.Unmarshal([]byte(test.body), &want) != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("body = %q, want %s", data, test.body)
	}
}

func readAppApplicationFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "app-application", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func withoutYes(arguments []string) []string {
	result := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if argument != "--yes" {
			result = append(result, argument)
		}
	}
	return result
}
