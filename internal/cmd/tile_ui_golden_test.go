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
	"github.com/spf13/cobra"
)

const (
	tileUIEnterpriseID = "11111111-1111-4111-8111-111111111111"
	tileUIWallpaperID  = "22222222-2222-4222-8222-222222222222"
	tileUIIconID       = "44444444-4444-4444-8444-444444444444"
	tileUIApplyBody    = `{"icon":"44444444-4444-4444-8444-444444444444","apply_mode":"devices","devices":["device-1"],"groups":null}`
)

type tileUIFixture struct {
	id, name, method, path, body string
	args                         []string
	query                        url.Values
	status, errorStatus          int
	all, multipart, destructive  bool
}

type tileUIOperationMetadata struct {
	method, path, noun, verb, pagination string
	destructive                          bool
}

func tileUIFixtures(t *testing.T) []tileUIFixture {
	t.Helper()
	image := filepath.Join(t.TempDir(), "wallpaper.png")
	icon := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(image, []byte("wallpaper-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(icon, []byte("icon-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	return []tileUIFixture{
		{"getWallpapers", "wallpaper-list", http.MethodGet, "/v1/enterprise/" + tileUIEnterpriseID + "/wallpaper/", "", []string{"wallpaper", "list", "--enterprise", tileUIEnterpriseID, "--created-on-gt", "2026-08-01T00:00:00Z", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"created_on_gt": {"2026-08-01T00:00:00Z"}, "limit": {"1"}, "offset": {"0"}}, 200, 400, true, false, false},
		{"addWallpaper", "wallpaper-add", http.MethodPost, "/v1/enterprise/" + tileUIEnterpriseID + "/wallpaper/", "", []string{"wallpaper", "add", "--enterprise", tileUIEnterpriseID, "--image-file", image, "--orientation", "landscape", "--json"}, nil, 201, 400, false, true, false},
		{"getWallpaper", "wallpaper-get", http.MethodGet, "/v1/enterprise/" + tileUIEnterpriseID + "/wallpaper/" + tileUIWallpaperID + "/", "", []string{"wallpaper", "get", tileUIEnterpriseID, tileUIWallpaperID, "--json"}, nil, 200, 404, false, false, false},
		{"deleteWallpaper", "wallpaper-delete", http.MethodDelete, "/v1/enterprise/" + tileUIEnterpriseID + "/wallpaper/" + tileUIWallpaperID + "/", "", []string{"wallpaper", "delete", tileUIEnterpriseID, tileUIWallpaperID, "--yes"}, nil, 204, 400, false, false, true},
		{"getTileIcons", "tile-icon-list", http.MethodGet, "/v1/enterprise/" + tileUIEnterpriseID + "/tile-icons/", "", []string{"tile-icon", "list", "--enterprise", tileUIEnterpriseID, "--category", "kiosk", "--custom", "true", "--limit", "1", "--offset", "0", "--all", "--json"}, url.Values{"category": {"kiosk"}, "custom": {"true"}, "limit": {"1"}, "offset": {"0"}}, 200, 400, true, false, false},
		{"addTileIcon", "tile-icon-add", http.MethodPost, "/v1/enterprise/" + tileUIEnterpriseID + "/tile-icons/", "", []string{"tile-icon", "add", "--enterprise", tileUIEnterpriseID, "--icon-file", icon, "--device-model", "TC57", "--category", "kiosk", "--custom", "true", "--json"}, nil, 201, 400, false, true, false},
		{"getTileIcon", "tile-icon-get", http.MethodGet, "/v1/enterprise/" + tileUIEnterpriseID + "/tile-icons/" + tileUIIconID + "/", "", []string{"tile-icon", "get", tileUIEnterpriseID, tileUIIconID, "--json"}, nil, 200, 404, false, false, false},
		{"deleteTileIcon", "tile-icon-delete", http.MethodDelete, "/v1/enterprise/" + tileUIEnterpriseID + "/tile-icons/" + tileUIIconID + "/", "", []string{"tile-icon", "delete", tileUIEnterpriseID, tileUIIconID, "--yes"}, nil, 204, 400, false, false, true},
		{"applyTileIcon", "tile-icon-apply", http.MethodPost, "/v1/enterprise/" + tileUIEnterpriseID + "/tile-icon-apply/", `{"apply_mode":"device_model","device_model":"TC57","icon":"44444444-4444-4444-8444-444444444444"}`, []string{"tile-icon-apply", "apply", tileUIEnterpriseID, "--icon", tileUIIconID, "--apply-mode", "device_model", "--device-model", "TC57", "--json"}, nil, 201, 400, false, false, false},
		{"removeTileIcon", "tile-icon-unapply", http.MethodPost, "/v1/enterprise/" + tileUIEnterpriseID + "/tile-icon-unapply/", `{"model":"TC57"}`, []string{"tile-icon-unapply", "remove", tileUIEnterpriseID, "--model", "TC57", "--yes", "--json"}, nil, 201, 400, false, false, true},
	}
}

func TestTileUIOperationInventory(t *testing.T) {
	want := map[string]tileUIOperationMetadata{
		"getWallpapers":   {http.MethodGet, "/v1/enterprise/{enterprise_id}/wallpaper/", "wallpaper", "list", "limit-offset", false},
		"addWallpaper":    {http.MethodPost, "/v1/enterprise/{enterprise_id}/wallpaper/", "wallpaper", "add", "none", false},
		"getWallpaper":    {http.MethodGet, "/v1/enterprise/{enterprise_id}/wallpaper/{wallpaper_id}/", "wallpaper", "get", "none", false},
		"deleteWallpaper": {http.MethodDelete, "/v1/enterprise/{enterprise_id}/wallpaper/{wallpaper_id}/", "wallpaper", "delete", "none", true},
		"getTileIcons":    {http.MethodGet, "/v1/enterprise/{enterprise_id}/tile-icons/", "tile-icon", "list", "limit-offset", false},
		"addTileIcon":     {http.MethodPost, "/v1/enterprise/{enterprise_id}/tile-icons/", "tile-icon", "add", "none", false},
		"getTileIcon":     {http.MethodGet, "/v1/enterprise/{enterprise_id}/tile-icons/{tileicons_id}/", "tile-icon", "get", "none", false},
		"deleteTileIcon":  {http.MethodDelete, "/v1/enterprise/{enterprise_id}/tile-icons/{tileicons_id}/", "tile-icon", "delete", "none", true},
		"applyTileIcon":   {http.MethodPost, "/v1/enterprise/{enterprise_id}/tile-icon-apply/", "tile-icon-apply", "apply", "none", false},
		"removeTileIcon":  {http.MethodPost, "/v1/enterprise/{enterprise_id}/tile-icon-unapply/", "tile-icon-unapply", "remove", "none", true},
	}
	got := map[string]tileUIOperationMetadata{}
	for _, operation := range generated.Operations() {
		if operation.Generation != "v1" {
			continue
		}
		if _, ok := want[operation.OperationID]; !ok {
			continue
		}
		got[operation.OperationID] = tileUIOperationMetadata{operation.Method, operation.Path, operation.Noun, operation.Verb, operation.Pagination, operation.Destructive}
	}
	if len(want) != 10 || !reflect.DeepEqual(want, got) {
		t.Fatalf("tile UI operation inventory = %#v", got)
	}
	if len(tileUIFixtures(t)) != 10 {
		t.Fatalf("fixture rows = %d, want 10", len(tileUIFixtures(t)))
	}
}

func TestTileUIFixtureInventory(t *testing.T) {
	want := map[string]bool{"README.md": true, "RERECORD_WITH_TENANT": true, "wallpaper-list-second-page.json": true, "tile-icon-list-second-page.json": true}
	for _, row := range tileUIFixtures(t) {
		want[row.name+"-success.json"] = true
		want[row.name+"-success.golden"] = true
		want[row.name+"-api-error.json"] = true
	}
	entries, err := os.ReadDir(filepath.Join("..", "..", "spec", "fixtures", "tile-ui"))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, entry := range entries {
		got[entry.Name()] = true
	}
	if len(want) != 34 || !reflect.DeepEqual(want, got) {
		t.Fatalf("tile UI fixture inventory = %#v, want 34 exact files", got)
	}
}

func TestTileUISchemasAndFixtures(t *testing.T) {
	document := tileUIDocument(t)
	wallpaperAdd := tileUIRequestSchema(t, tileUIOperation(document, "addWallpaper"), "multipart/form-data")
	if wallpaperAdd["properties"].(map[string]any)["image_file"].(map[string]any)["format"] != "binary" || !reflect.DeepEqual(wallpaperAdd["properties"].(map[string]any)["orientation"].(map[string]any)["enum"], []any{"portrait", "landscape"}) {
		t.Fatalf("wallpaper add schema = %#v", wallpaperAdd)
	}
	tileAdd := tileUIRequestSchema(t, tileUIOperation(document, "addTileIcon"), "multipart/form-data")
	tileAddProperties := tileAdd["properties"].(map[string]any)
	if tileAddProperties["icon_file"].(map[string]any)["format"] != "binary" || !reflect.DeepEqual(tileAddProperties["category"].(map[string]any)["enum"], []any{"mobile", "tablet", "rugged", "pos", "digital_signage", "kiosk"}) || tileAddProperties["custom"].(map[string]any)["type"] != "boolean" {
		t.Fatalf("tile-icon add schema = %#v", tileAdd)
	}
	applyOperation := tileUIOperation(document, "applyTileIcon")
	if applyOperation["requestBody"].(map[string]any)["required"] != true || tileUIRequestSchema(t, applyOperation, "application/json")["$ref"] != "#/components/schemas/ApplyTileIcon" {
		t.Fatalf("apply request schema = %#v", applyOperation["requestBody"])
	}
	unapplyOperation := tileUIOperation(document, "removeTileIcon")
	if unapplyOperation["requestBody"].(map[string]any)["required"] != true || !reflect.DeepEqual(tileUIRequestSchema(t, unapplyOperation, "application/json")["required"], []any{"model"}) || !reflect.DeepEqual(fixtureSchemaResponse(t, document, unapplyOperation, 201)["required"], []any{"model"}) {
		t.Fatalf("unapply schemas = request %#v response %#v", unapplyOperation["requestBody"], fixtureSchemaResponse(t, document, unapplyOperation, 201))
	}
	wallpaper := document["components"].(map[string]any)["schemas"].(map[string]any)["Wallpaper"].(map[string]any)
	if wallpaper["properties"].(map[string]any)["enterprise"].(map[string]any)["format"] != nil {
		t.Fatal("wallpaper enterprise must be a plain string")
	}
	for _, name := range []string{"thumbnail", "orientation"} {
		property := wallpaper["properties"].(map[string]any)[name].(map[string]any)
		if name == "thumbnail" && !reflect.DeepEqual(property["type"], []any{"string", "null"}) {
			t.Fatalf("thumbnail type = %#v", property["type"])
		}
		if name == "orientation" && !reflect.DeepEqual(property["enum"], []any{"portrait", "landscape"}) {
			t.Fatalf("orientation enum = %#v", property["enum"])
		}
	}
	apply := document["components"].(map[string]any)["schemas"].(map[string]any)["ApplyTileIcon"].(map[string]any)
	if !reflect.DeepEqual(apply["required"], []any{"icon"}) {
		t.Fatalf("apply required = %#v", apply["required"])
	}
	for name, max := range map[string]float64{"devices": 100, "groups": 10} {
		property := apply["properties"].(map[string]any)[name].(map[string]any)
		if property["maxItems"] != max || !reflect.DeepEqual(property["type"], []any{"array", "null"}) {
			t.Fatalf("%s = %#v", name, property)
		}
	}
	for _, row := range tileUIFixtures(t) {
		operation := tileUIOperation(document, row.id)
		tileUIValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-success.json", row.status != 204)
		tileUIValidateFixture(t, document, fixtureSchemaResponseForStatus(document, operation, row.errorStatus), row.name+"-api-error.json", true)
		if row.all {
			tileUIValidateFixture(t, document, fixtureSchemaResponse(t, document, operation, row.status), row.name+"-second-page.json", true)
		}
	}
}

func TestTileUIGoldenFixtures(t *testing.T) {
	for _, row := range tileUIFixtures(t) {
		t.Run(row.name, func(t *testing.T) { executeTileUIFixture(t, row, false) })
	}
}

func TestTileUIAPIErrors(t *testing.T) {
	for _, row := range tileUIFixtures(t) {
		t.Run(row.name, func(t *testing.T) { executeTileUIFixture(t, row, true) })
	}
}

func TestTileUIInputRules(t *testing.T) {
	root := NewRootCommand()
	add, _, _ := root.Find([]string{"tile-icon", "add"})
	if add.Flags().Lookup("enterprise") == nil || add.Flags().Lookup("icon-file") == nil || add.Flags().Lookup("category") == nil || add.Flags().Lookup("custom") == nil {
		t.Fatal("tile-icon add flags are incomplete")
	}
	if add.Flags().Lookup("enterprise").Usage != "scope routes to /v1/enterprise/{enterprise_id}/tile-icons/" {
		t.Fatal("tile-icon enterprise must be the scope flag, not a body property flag")
	}
	for _, operation := range generated.Operations() {
		if operation.OperationID == "addTileIcon" && (operation.Body == nil || len(operation.Body.AutoFill) != 1 || operation.Body.AutoFill[0].Name != "enterprise" || operation.Body.AutoFill[0].Format != "uuid") {
			t.Fatalf("tile-icon add auto-fill = %#v", operation.Body)
		}
	}
	apply, _, _ := root.Find([]string{"tile-icon-apply", "apply"})
	for _, name := range []string{"icon", "apply-mode", "device-model", "body"} {
		if apply.Flags().Lookup(name) == nil {
			t.Fatalf("tile-icon apply missing --%s", name)
		}
	}
	for _, name := range []string{"devices", "groups"} {
		if apply.Flags().Lookup(name) != nil {
			t.Fatalf("tile-icon apply exposes complex --%s", name)
		}
	}
	for _, args := range [][]string{
		{"wallpaper", "add", "--enterprise", tileUIEnterpriseID},
		{"tile-icon", "add", "--enterprise", tileUIEnterpriseID},
		{"tile-icon-apply", "apply", tileUIEnterpriseID},
		{"tile-icon-apply", "apply", tileUIEnterpriseID, "--apply-mode", "all_devices"},
		{"tile-icon-apply", "apply", tileUIEnterpriseID, "--body", "{"},
		{"tile-icon-apply", "apply", tileUIEnterpriseID, "--body", tileUIApplyBody, "--icon", tileUIIconID},
		{"tile-icon-unapply", "remove", tileUIEnterpriseID},
	} {
		assertTileUIUsage(t, args)
	}
	rows := tileUIFixtures(t)
	for _, args := range [][]string{
		withoutTileUIFlag(rows[1].args, "--image-file"),
		withoutTileUIFlag(rows[1].args, "--orientation"),
		withoutTileUIFlag(rows[5].args, "--icon-file"),
		withoutTileUIFlag(rows[5].args, "--device-model"),
	} {
		assertTileUIUsage(t, args)
	}
	for _, mode := range []string{"inline", "file", "stdin"} {
		assertTileUIApplyBodyMode(t, mode)
	}
	missingIcon := tileUIFixtures(t)[8]
	missingIcon.args = []string{"tile-icon-apply", "apply", tileUIEnterpriseID, "--body", `{"apply_mode":"all_devices"}`, "--json"}
	missingIcon.body = `{"apply_mode":"all_devices"}`
	executeTileUIFixture(t, missingIcon, true)
}

func TestTileUIDestructiveCommandsDeclined(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	for _, row := range tileUIFixtures(t) {
		if !row.destructive {
			continue
		}
		command := configuredTileUICommand(t, server.URL)
		command.SetIn(strings.NewReader("no\n"))
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)
		command.SetArgs(withoutYes(row.args))
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) = %v", row.args, err)
		}
	}
	if requests != 0 {
		t.Fatalf("declined destructive requests = %d, want 0", requests)
	}
}

func executeTileUIFixture(t *testing.T, row tileUIFixture, apiError bool) {
	t.Helper()
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != row.method || request.URL.Path != row.path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, row.method, row.path)
		}
		for _, name := range []string{"X-Esper-Tenant-ID", "X-Esper-Caller-ID", "X-Esper-User-ID", "X-Esper-Private-Service"} {
			if request.Header.Get(name) != "" {
				t.Errorf("internal header %s was sent", name)
			}
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" || request.Header.Get("Accept") != "application/json" {
			t.Errorf("headers authorization=%q accept=%q", request.Header.Get("Authorization"), request.Header.Get("Accept"))
		}
		wantQuery := row.query
		fixture := row.name + "-success.json"
		if row.all && !apiError && requests == 2 {
			wantQuery = tileUISecondPageQuery(row)
			fixture = row.name + "-second-page.json"
		}
		if got := request.URL.Query().Encode(); got != wantQuery.Encode() {
			t.Errorf("query = %q, want %q", got, wantQuery.Encode())
		}
		tileUIAssertBody(t, request, row)
		status := row.status
		if apiError {
			fixture, status = row.name+"-api-error.json", row.errorStatus
		}
		response := readTileUIFixture(t, fixture)
		if row.all && !apiError && requests == 1 {
			response = bytes.Replace(response, []byte("NEXT_PAGE"), []byte(server.URL+row.path+"?"+tileUISecondPageQuery(row).Encode()), 1)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(response)
	}))
	defer server.Close()
	args := append([]string(nil), row.args...)
	if apiError && row.all {
		args = append(args[:len(args)-2], "--json")
	}
	command := configuredTileUICommand(t, server.URL)
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs(args)
	err := command.Execute()
	if apiError {
		var api *esperruntime.APIError
		if !errors.As(err, &api) || api.StatusCode != row.errorStatus || esperruntime.ExitCode(err) != 1 || stdout.Len() != 0 || stderr.Len() != 0 || !bytes.Equal(api.Body, readTileUIFixture(t, row.name+"-api-error.json")) {
			t.Fatalf("API error=%v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
		}
		return
	}
	if err != nil || stderr.Len() != 0 || !bytes.Equal(stdout.Bytes(), readTileUIFixture(t, row.name+"-success.golden")) {
		t.Fatalf("error=%v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
	}
	if row.all && requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func tileUISecondPageQuery(row tileUIFixture) url.Values {
	query := make(url.Values, len(row.query))
	for name, values := range row.query {
		query[name] = append([]string(nil), values...)
	}
	query.Set("offset", "1")
	return query
}

func tileUIAssertBody(t *testing.T, request *http.Request, row tileUIFixture) {
	t.Helper()
	if row.multipart {
		mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Fatalf("content type = %q, error = %v", request.Header.Get("Content-Type"), err)
		}
		if err := request.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		fileName, fileBytes := "image_file", "wallpaper-bytes"
		if row.id == "addTileIcon" {
			fileName, fileBytes = "icon_file", "icon-bytes"
			if values := request.MultipartForm.Value["enterprise"]; !reflect.DeepEqual(values, []string{tileUIEnterpriseID}) {
				t.Fatalf("enterprise multipart values = %#v", values)
			}
			for name, want := range map[string]string{"device_model": "TC57", "category": "kiosk", "custom": "true"} {
				if values := request.MultipartForm.Value[name]; !reflect.DeepEqual(values, []string{want}) {
					t.Fatalf("%s multipart values = %#v", name, values)
				}
			}
		} else if values := request.MultipartForm.Value["orientation"]; !reflect.DeepEqual(values, []string{"landscape"}) {
			t.Fatalf("orientation multipart values = %#v", values)
		}
		files := request.MultipartForm.File[fileName]
		if len(files) != 1 {
			t.Fatalf("%s files = %d", fileName, len(files))
		}
		file, err := files[0].Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil || string(data) != fileBytes {
			t.Fatalf("%s = %q, error = %v", fileName, data, err)
		}
		return
	}
	data, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if row.body == "" {
		if len(data) != 0 {
			t.Errorf("body = %q, want empty", data)
		}
		return
	}
	if request.Header.Get("Content-Type") != "application/json" || string(data) != row.body {
		t.Errorf("body=%q content-type=%q, want %q", data, request.Header.Get("Content-Type"), row.body)
	}
}

func assertTileUIApplyBodyMode(t *testing.T, mode string) {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		data, _ := io.ReadAll(request.Body)
		if request.Method != http.MethodPost || request.URL.Path != "/v1/enterprise/"+tileUIEnterpriseID+"/tile-icon-apply/" || string(data) != tileUIApplyBody || request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("request=%s %s body=%q content-type=%q", request.Method, request.URL.Path, data, request.Header.Get("Content-Type"))
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write(readTileUIFixture(t, "tile-icon-apply-success.json"))
	}))
	defer server.Close()
	value, input := tileUIApplyBody, io.Reader(strings.NewReader(""))
	if mode == "file" {
		file := filepath.Join(t.TempDir(), "apply.json")
		if err := os.WriteFile(file, []byte(tileUIApplyBody), 0o600); err != nil {
			t.Fatal(err)
		}
		value = "@" + file
	} else if mode == "stdin" {
		value, input = "-", strings.NewReader(tileUIApplyBody)
	}
	command := configuredTileUICommand(t, server.URL)
	command.SetIn(input)
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"tile-icon-apply", "apply", tileUIEnterpriseID, "--body", value})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute(%s): %v", mode, err)
	}
}

func assertTileUIUsage(t *testing.T, args []string) {
	t.Helper()
	command := NewRootCommand()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("Execute(%v) = %v", args, err)
	}
}

func withoutTileUIFlag(args []string, name string) []string {
	result := make([]string, 0, len(args)-2)
	for index := 0; index < len(args); index++ {
		if args[index] == name {
			index++
			continue
		}
		result = append(result, args[index])
	}
	return result
}

func configuredTileUICommand(t *testing.T, endpoint string) *cobra.Command {
	t.Helper()
	t.Setenv(esperruntime.EnvironmentVariable, endpoint)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")
	return NewRootCommand()
}

func tileUIDocument(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "openapi", "v1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func tileUIOperation(document map[string]any, operationID string) map[string]any {
	for _, path := range document["paths"].(map[string]any) {
		for _, operation := range path.(map[string]any) {
			if value, ok := operation.(map[string]any); ok && value["operationId"] == operationID {
				return value
			}
		}
	}
	panic("operation not found: " + operationID)
}

func tileUIRequestSchema(t *testing.T, operation map[string]any, mediaType string) map[string]any {
	t.Helper()
	content := operation["requestBody"].(map[string]any)["content"].(map[string]any)
	schema, ok := content[mediaType].(map[string]any)["schema"].(map[string]any)
	if !ok {
		t.Fatalf("request media %s missing", mediaType)
	}
	return schema
}

func tileUIValidateFixture(t *testing.T, document, schema map[string]any, name string, jsonBody bool) {
	t.Helper()
	data := readTileUIFixture(t, name)
	if !jsonBody {
		if len(data) != 0 {
			t.Fatalf("%s = %q, want empty", name, data)
		}
		return
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	if err := fixtureSchemaValidate(document, schema, value, "$", map[string]bool{}); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

func readTileUIFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "tile-ui", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
