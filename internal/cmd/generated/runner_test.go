package generated

import (
	"bytes"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

func TestScopedCollectionUsesParentFlag(t *testing.T) {
	operations := []Operation{
		{Path: "/pipelines/v0/runs/", ScopeParent: ""},
		{Path: "/pipelines/v0/pipelines/{pipeline_id}/runs/", ScopeParent: "pipeline", Parameters: []Parameter{{Name: "pipeline_id", In: "path", Required: true, Scope: true}}},
	}
	command := &cobra.Command{Use: "list"}
	addFlags(command, operations)
	if command.Flags().Lookup("pipeline") == nil {
		t.Fatal("scoped collection did not expose --pipeline")
	}
	if command.Flags().Lookup("pipeline-id") != nil {
		t.Fatal("scoped collection exposed --pipeline-id instead of --pipeline")
	}
	if err := command.Flags().Set("pipeline", "pipeline-1"); err != nil {
		t.Fatal(err)
	}
	selected, err := selectOperation(command, operations)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Path != "/pipelines/v0/pipelines/{pipeline_id}/runs/" {
		t.Fatalf("selected %s", selected.Path)
	}
}

func TestRequiredBodies(t *testing.T) {
	tests := []struct {
		name      string
		body      *Body
		set       map[string]string
		wantBody  string
		wantUsage bool
	}{
		{name: "nonempty needs input", body: &Body{MediaType: "application/json", Required: true, Properties: []Property{{Name: "name", Type: "string"}}}, wantUsage: true},
		{name: "nonempty accepts optional input", body: &Body{MediaType: "application/json", Required: true, Properties: []Property{{Name: "name", Type: "string"}}}, set: map[string]string{"name": "kiosk"}, wantBody: `{"name":"kiosk"}`},
		{name: "empty sends object", body: &Body{MediaType: "application/json", Required: true, Empty: true}, wantBody: `{}`},
		{name: "optional body sends nothing", body: &Body{MediaType: "application/json", Properties: []Property{{Name: "name", Type: "string"}}}, wantBody: ""},
		{name: "complex requires raw body", body: &Body{MediaType: "application/json", BodyOnly: true}, wantUsage: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := &cobra.Command{Use: "create"}
			operation := Operation{Body: test.body}
			addFlags(command, []Operation{operation})
			for name, value := range test.set {
				if err := command.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}
			body, _, err := bodyFor(command, operation)
			if test.wantUsage {
				if err == nil || esperruntime.ExitCode(err) != 2 {
					t.Fatalf("bodyFor() error = %v", err)
				}
				return
			}
			if err != nil || string(body) != test.wantBody {
				t.Fatalf("bodyFor() = %s, %v", body, err)
			}
		})
	}
}

func TestRequiredEmptyMultipartSendsForm(t *testing.T) {
	command := &cobra.Command{Use: "upload"}
	operation := Operation{Body: &Body{MediaType: "multipart/form-data", Required: true, Empty: true}}
	addFlags(command, []Operation{operation})
	body, contentType, err := bodyFor(command, operation)
	if err != nil || len(body) == 0 || !strings.HasPrefix(contentType, "multipart/form-data") {
		t.Fatalf("bodyFor() = %q, %q, %v", body, contentType, err)
	}
}

func TestRequiredParametersAreConditionalOnSelectedRoute(t *testing.T) {
	operations := []Operation{
		{Path: "/items", Parameters: []Parameter{{Name: "app_id", In: "query", Required: true}}},
		{Path: "/devices/{device_id}/items", ScopeParent: "device", Parameters: []Parameter{{Name: "device_id", In: "path", Scope: true, ScopeName: "device", Required: true}}},
	}
	command := &cobra.Command{Use: "list"}
	addFlags(command, operations)
	selected, err := selectOperation(command, operations)
	if err != nil || validateRequiredParameters(command, selected) == nil {
		t.Fatalf("unscoped required query was not enforced: %v", err)
	}
	if err := command.Flags().Set("device", "device-1"); err != nil {
		t.Fatal(err)
	}
	selected, err = selectOperation(command, operations)
	if err != nil || validateRequiredParameters(command, selected) != nil {
		t.Fatalf("scoped route inherited unscoped requirement: %v", err)
	}
}

func TestRequireOneOfParameters(t *testing.T) {
	operation := Operation{
		RequireOneOf: []string{"request", "device"},
		Parameters: []Parameter{
			{Name: "request", In: "query"},
			{Name: "device", In: "query"},
		},
	}
	command := &cobra.Command{Use: "list"}
	addFlags(command, []Operation{operation})
	if err := validateRequiredParameters(command, operation); err == nil || esperruntime.ExitCode(err) != 2 || !strings.Contains(err.Error(), "--request, --device") {
		t.Fatalf("missing one-of error = %v", err)
	}
	if usage := command.Flags().Lookup("request").Usage; !strings.Contains(usage, "at least one required") {
		t.Fatalf("--request usage = %q", usage)
	}
	if err := command.Flags().Set("device", "device-1"); err != nil {
		t.Fatal(err)
	}
	if err := validateRequiredParameters(command, operation); err != nil {
		t.Fatalf("one-of validation = %v", err)
	}
}

func TestRequiredParameterHelp(t *testing.T) {
	operation := Operation{Parameters: []Parameter{{Name: "parent_group_ids", In: "query", Required: true}}}
	command := &cobra.Command{Use: "list"}
	addFlags(command, []Operation{operation})
	if usage := command.Flags().Lookup("parent-group-ids").Usage; usage != "parent-group-ids (required)" {
		t.Fatalf("required flag usage = %q", usage)
	}
}

func TestRequiredBodyPropertiesAreConditionalOnBodyInput(t *testing.T) {
	tests := []struct {
		name      string
		required  bool
		set       map[string]string
		wantUsage bool
	}{
		{name: "required body enforces every required property", required: true, set: map[string]string{"description": "fixture"}, wantUsage: true},
		{name: "optional omitted body needs no properties"},
		{name: "optional supplied body enforces required properties", set: map[string]string{"description": "fixture"}, wantUsage: true},
		{name: "raw body skips property enforcement", required: true, set: map[string]string{"body": `{"nested":true}`}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation := Operation{Method: "PATCH", Body: &Body{MediaType: "application/json", Required: test.required, Properties: []Property{{Name: "name", Type: "string", Required: true}, {Name: "description", Type: "string"}}}}
			command := &cobra.Command{Use: "patch"}
			addFlags(command, []Operation{operation})
			for name, value := range test.set {
				if err := command.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}
			_, _, err := bodyFor(command, operation)
			if test.wantUsage {
				if err == nil || esperruntime.ExitCode(err) != 2 {
					t.Fatalf("bodyFor() error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCollisionAutoFill(t *testing.T) {
	operation := Operation{
		Parameters: []Parameter{{Name: "enterprise_id", In: "path", Scope: true, ScopeName: "enterprise"}},
		Body:       &Body{MediaType: "application/json", Required: true, AutoFill: []AutoFill{{Name: "enterprise", Parameter: "enterprise_id", Type: "string"}}},
	}
	command := &cobra.Command{Use: "create"}
	addFlags(command, []Operation{operation})
	if command.Flags().Lookup("enterprise") == nil {
		t.Fatal("scope flag missing")
	}
	if err := command.Flags().Set("enterprise", "enterprise-1"); err != nil {
		t.Fatal(err)
	}
	body, _, err := bodyForValues(command, operation, map[string]string{"enterprise_id": "enterprise-1"})
	if err != nil || string(body) != `{"enterprise":"enterprise-1"}` {
		t.Fatalf("bodyForValues() = %s, %v", body, err)
	}
	if err := command.Flags().Set("body", `{"enterprise":"wrong"}`); err != nil {
		t.Fatal(err)
	}
	body, _, err = bodyForValues(command, operation, map[string]string{"enterprise_id": "enterprise-1"})
	if err != nil || string(body) != `{"enterprise":"enterprise-1"}` {
		t.Fatalf("raw body auto-fill = %s, %v", body, err)
	}

	multipartOperation := operation
	multipartOperation.Body = &Body{MediaType: "multipart/form-data", Required: true, AutoFill: operation.Body.AutoFill}
	body, contentType, err := bodyForValues(command, multipartOperation, map[string]string{"enterprise_id": "enterprise-1"})
	if err != nil {
		t.Fatal(err)
	}
	_, params, _ := mime.ParseMediaType(contentType)
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "enterprise" {
		t.Fatalf("multipart auto-fill part = %v, %v", part, err)
	}
}

func TestURLCollisionAutoFillUsesResourceURL(t *testing.T) {
	operation := Operation{
		Path:       "/enterprise/{enterprise_id}/policy/",
		Parameters: []Parameter{{Name: "enterprise_id", In: "path", Scope: true, ScopeName: "enterprise"}},
		Body:       &Body{MediaType: "application/json", Required: true, BodyOnly: true, AutoFill: []AutoFill{{Name: "enterprise", Parameter: "enterprise_id", Type: "string", Format: "url"}}},
	}
	command := &cobra.Command{Use: "create"}
	addFlags(command, []Operation{operation})
	if err := command.Flags().Set("enterprise", "enterprise-1"); err != nil {
		t.Fatal(err)
	}
	if err := command.Flags().Set("body", `{"enterprise":"https://wrong.invalid/","policy":{}}`); err != nil {
		t.Fatal(err)
	}
	body, _, err := bodyForValues(command, operation, map[string]string{"enterprise_id": "enterprise-1"})
	if err != nil || string(body) != `{"enterprise":"/enterprise/enterprise-1/","policy":{}}` {
		t.Fatalf("bodyForValues() = %s, %v", body, err)
	}
	body, err = qualifyAutoFillURLs(body, operation.Body, "https://develop-api.esper.cloud/api")
	if err != nil || string(body) != `{"enterprise":"https://develop-api.esper.cloud/api/enterprise/enterprise-1/","policy":{}}` {
		t.Fatalf("qualified body = %s, %v", body, err)
	}
}

func TestOptionalBodyAutoFillRequiresBodyInput(t *testing.T) {
	operation := Operation{
		Parameters: []Parameter{{Name: "deviceId", In: "path", Scope: true, ScopeName: "device"}},
		Body:       &Body{MediaType: "application/json", AutoFill: []AutoFill{{Name: "device_id", Parameter: "deviceId", Type: "string"}}},
	}
	command := &cobra.Command{Use: "create"}
	addFlags(command, []Operation{operation})
	pathValues := map[string]string{"deviceId": "device-1"}
	body, _, err := bodyForValues(command, operation, pathValues)
	if err != nil || body != nil {
		t.Fatalf("omitted optional body = %s, %v", body, err)
	}
	if err := command.Flags().Set("body", `{"device_id":"wrong"}`); err != nil {
		t.Fatal(err)
	}
	body, _, err = bodyForValues(command, operation, pathValues)
	if err != nil || string(body) != `{"device_id":"device-1"}` {
		t.Fatalf("optional raw body auto-fill = %s, %v", body, err)
	}
}

func TestRecursiveScopesLeaveResourceIDPositional(t *testing.T) {
	operation := Operation{Path: "/stages/{stage_id}/runs/{run_id}/commands/{command_id}", ScopeParent: "run", Parameters: []Parameter{
		{Name: "stage_id", In: "path", Scope: true, ScopeName: "stage"},
		{Name: "run_id", In: "path", Scope: true, ScopeName: "run"},
		{Name: "command_id", In: "path", Required: true},
	}}
	command := &cobra.Command{Use: "get"}
	addFlags(command, []Operation{operation})
	_ = command.Flags().Set("stage", "stage-1")
	_ = command.Flags().Set("run", "run-1")
	path, err := replacePath(command, operation, []string{"command-1"})
	if err != nil || path != "/stages/stage-1/runs/run-1/commands/command-1" {
		t.Fatalf("replacePath() = %q, %v", path, err)
	}
}

func TestRecursiveScopesSelectExactRoute(t *testing.T) {
	operations := []Operation{
		{Path: "/runs/{run_id}/commands", Parameters: []Parameter{{Name: "run_id", In: "path", Scope: true, ScopeName: "run"}}},
		{Path: "/stages/{stage_id}/runs/{run_id}/commands", Parameters: []Parameter{{Name: "stage_id", In: "path", Scope: true, ScopeName: "stage"}, {Name: "run_id", In: "path", Scope: true, ScopeName: "run"}}},
	}
	command := &cobra.Command{Use: "list"}
	addFlags(command, operations)
	_ = command.Flags().Set("run", "run-1")
	selected, err := selectOperation(command, operations)
	if err != nil || selected.Path != operations[0].Path {
		t.Fatalf("run route = %#v, %v", selected, err)
	}
	_ = command.Flags().Set("stage", "stage-1")
	selected, err = selectOperation(command, operations)
	if err != nil || selected.Path != operations[1].Path {
		t.Fatalf("nested route = %#v, %v", selected, err)
	}
}

func TestAliasesDoNotCreateRoutes(t *testing.T) {
	operations := []Operation{
		{Command: []string{"geofence", "get"}, Path: "/geofence/{id}", OperationID: "canonical"},
		{Command: []string{"geofence", "get"}, Path: "/the-geofence/{id}", OperationID: "alias", AliasOf: "canonical"},
	}
	groups := executableOperationGroups(operations)
	group := groups["geofence\x00get"]
	if len(groups) != 1 || len(group) != 1 || group[0].OperationID != "canonical" {
		t.Fatalf("executable groups = %#v", groups)
	}
}

func TestNonJSONOutput(t *testing.T) {
	command := &cobra.Command{Use: "get"}
	command.Flags().String("output", "", "")
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	if err := writeRawResponse(command, []byte("plist")); err != nil || stdout.String() != "plist" {
		t.Fatalf("stdout response = %q, %v", stdout.String(), err)
	}
	path := filepath.Join(t.TempDir(), "response.plist")
	if err := command.Flags().Set("output", path); err != nil {
		t.Fatal(err)
	}
	if err := writeRawResponse(command, []byte("file")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "file" {
		t.Fatalf("output file = %q, %v", data, err)
	}
	if !strings.Contains("application/x-plist", "plist") || isJSONMedia("application/x-plist") {
		t.Fatal("non-JSON media detection failed")
	}
}

func TestJSONBodyRules(t *testing.T) {
	operation := Operation{Body: &Body{MediaType: "application/json", Properties: []Property{{Name: "name", Type: "string", Required: true}, {Name: "enabled", Type: "boolean"}}}}
	command := &cobra.Command{Use: "create"}
	addFlags(command, []Operation{operation})
	if err := command.Flags().Set("body", `{"nested":{"id":"one"}}`); err != nil {
		t.Fatal(err)
	}
	body, _, err := bodyFor(command, operation)
	if err != nil || string(body) != `{"nested":{"id":"one"}}` {
		t.Fatalf("bodyFor() = %s, %v", body, err)
	}
	if err := command.Flags().Set("name", "kiosk"); err != nil {
		t.Fatal(err)
	}
	_, _, err = bodyFor(command, operation)
	if err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("bodyFor() error = %v", err)
	}
}

func TestMultiParentScopesRequireExactlyOneWithoutGlobalRoute(t *testing.T) {
	operations := []Operation{
		{Path: "/blueprints/{blueprint_id}/versions", ScopeParent: "blueprint", Parameters: []Parameter{{Name: "blueprint_id", In: "path", Scope: true, ScopeName: "blueprint"}}},
		{Path: "/tenant-apps/{app_id}/versions", ScopeParent: "tenant-app", Parameters: []Parameter{{Name: "app_id", In: "path", Scope: true, ScopeName: "tenant-app"}}},
	}
	command := &cobra.Command{Use: "list"}
	addFlags(command, operations)
	if command.Flags().Lookup("blueprint") == nil || command.Flags().Lookup("tenant-app") == nil {
		t.Fatal("missing multi-parent scope flags")
	}
	_, err := selectOperation(command, operations)
	if err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("selectOperation() error = %v", err)
	}
	if err := command.Flags().Set("blueprint", "blueprint-1"); err != nil {
		t.Fatal(err)
	}
	selected, err := selectOperation(command, operations)
	if err != nil || selected.ScopeParent != "blueprint" {
		t.Fatalf("selectOperation() = %#v, %v", selected, err)
	}
	if err := command.Flags().Set("tenant-app", "app-1"); err != nil {
		t.Fatal(err)
	}
	_, err = selectOperation(command, operations)
	if err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("selectOperation() error = %v", err)
	}
}

func TestPathContextFallback(t *testing.T) {
	operation := Operation{Path: "/devices/{device_id}", Parameters: []Parameter{{Name: "device_id", In: "path", Required: true}}}
	command := &cobra.Command{Use: "get"}
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	active := esperruntime.ActiveContext{Device: &esperruntime.ActiveResource{ID: "device-1"}}
	values, err := resolvedPathValuesWithContext(command, operation, nil, active, true)
	if err != nil || values["device_id"] != "device-1" {
		t.Fatalf("resolvedPathValuesWithContext() = %#v, %v", values, err)
	}
	if stderr.String() != "context: using active device device-1 for device_id\n" {
		t.Fatalf("verbose output = %q", stderr.String())
	}

	values, err = resolvedPathValuesWithContext(command, operation, []string{"explicit-device"}, active, true)
	if err != nil || values["device_id"] != "explicit-device" {
		t.Fatalf("explicit path value = %#v, %v", values, err)
	}
}

func TestScopeContextFallback(t *testing.T) {
	operations := []Operation{{Path: "/enterprise/{enterprise_id}/apps", Parameters: []Parameter{{Name: "enterprise_id", In: "path", Required: true, Scope: true, ScopeName: "enterprise"}}}}
	command := &cobra.Command{Use: "list"}
	addFlags(command, operations)
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	active := esperruntime.ActiveContext{Enterprise: &esperruntime.ActiveResource{ID: "enterprise-1"}}
	if err := applyScopeContextFallbacks(command, operations, active, true); err != nil {
		t.Fatal(err)
	}
	selected, err := selectOperation(command, operations)
	if err != nil || selected.Path != operations[0].Path {
		t.Fatalf("selectOperation() = %#v, %v", selected, err)
	}
	if value, _ := command.Flags().GetString("enterprise"); value != "enterprise-1" {
		t.Fatalf("--enterprise = %q", value)
	}
	if stderr.String() != "context: using active enterprise enterprise-1 for enterprise_id\n" {
		t.Fatalf("verbose output = %q", stderr.String())
	}
}

func TestContextFallbackDoesNotOverrideGlobalOrExplicitScope(t *testing.T) {
	operations := []Operation{
		{Path: "/apps"},
		{Path: "/enterprise/{enterprise_id}/apps", Parameters: []Parameter{{Name: "enterprise_id", In: "path", Required: true, Scope: true, ScopeName: "enterprise"}}},
	}
	command := &cobra.Command{Use: "list"}
	addFlags(command, operations)
	active := esperruntime.ActiveContext{Enterprise: &esperruntime.ActiveResource{ID: "active-enterprise"}}
	if err := applyScopeContextFallbacks(command, operations, active, false); err != nil {
		t.Fatal(err)
	}
	selected, err := selectOperation(command, operations)
	if err != nil || selected.Path != "/apps" {
		t.Fatalf("global route = %#v, %v", selected, err)
	}

	if err := command.Flags().Set("enterprise", "explicit-enterprise"); err != nil {
		t.Fatal(err)
	}
	if err := applyScopeContextFallbacks(command, operations, active, false); err != nil {
		t.Fatal(err)
	}
	value, _ := command.Flags().GetString("enterprise")
	if value != "explicit-enterprise" {
		t.Fatalf("explicit scope was replaced with %q", value)
	}
}

func TestMissingContextErrorsIncludeSetHint(t *testing.T) {
	operation := Operation{Path: "/devices/{device_id}", Parameters: []Parameter{{Name: "device_id", In: "path", Required: true}}}
	command := &cobra.Command{Use: "get"}
	_, err := resolvedPathValuesWithContext(command, operation, nil, esperruntime.ActiveContext{}, false)
	if err == nil || !strings.Contains(err.Error(), "espercli context set device <id>") {
		t.Fatalf("missing path error = %v", err)
	}

	scoped := []Operation{{Path: "/enterprise/{enterprise_id}/apps", Parameters: []Parameter{{Name: "enterprise_id", In: "path", Required: true, Scope: true, ScopeName: "enterprise"}}}}
	addFlags(command, scoped)
	_, selectionErr := selectOperation(command, scoped)
	err = addScopeContextHint(command, scoped, esperruntime.ActiveContext{}, selectionErr)
	if err == nil || !strings.Contains(err.Error(), "espercli context set enterprise <id>") {
		t.Fatalf("missing scope error = %v", err)
	}
}

func TestCommandUseIncludesPositionalPathParameters(t *testing.T) {
	operations := []Operation{{Path: "/enterprise/{enterprise_id}/devices/{deviceId}/events/{event_id}", Parameters: []Parameter{
		{Name: "event_id", In: "path", Required: true},
		{Name: "deviceId", In: "path", Required: true},
		{Name: "enterprise_id", In: "path", Scope: true, ScopeName: "enterprise"},
		{Name: "limit", In: "query"},
	}}}
	if got, want := commandUse("get", operations), "get <device-id> <event-id>"; got != want {
		t.Fatalf("commandUse() = %q, want %q", got, want)
	}
}

func TestResolvedPathValuesFollowPathOrder(t *testing.T) {
	operation := Operation{Path: "/devices/{device_id}/events/{event_id}", Parameters: []Parameter{
		{Name: "event_id", In: "path", Required: true},
		{Name: "device_id", In: "path", Required: true},
	}}
	command := &cobra.Command{Use: "get"}
	values, err := resolvedPathValues(command, operation, []string{"device-1", "event-1"})
	if err != nil {
		t.Fatal(err)
	}
	if values["device_id"] != "device-1" || values["event_id"] != "event-1" {
		t.Fatalf("resolved values = %#v", values)
	}
}

func TestMergedCommandUseFollowsPrimaryRoute(t *testing.T) {
	operations := []Operation{
		{Verb: "get", Parameters: []Parameter{{Name: "operation_id", In: "path"}}},
		{Verb: "get", Parameters: []Parameter{{Name: "stage_run_operation_id", In: "path"}}},
	}
	if got, want := commandUse("get", operations), "get <operation-id>"; got != want {
		t.Fatalf("commandUse() = %q, want %q", got, want)
	}
}

func TestCommandLongHelpListsOlderAPIGenerations(t *testing.T) {
	operations := []Operation{{Command: []string{"application", "list"}, Noun: "application", Verb: "list"}}
	help := commandLongHelp("List applications", operations)
	if !strings.Contains(help, "Other API generations:") || !strings.Contains(help, "espercli api legacy application list") {
		t.Fatalf("commandLongHelp() = %q", help)
	}
}
