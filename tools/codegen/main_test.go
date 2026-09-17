package main

import "testing"

func TestExtractBodyRetainsRequirednessAndCollisionAutoFill(t *testing.T) {
	body := extractBody(map[string]requestMedia{
		"application/json": {Schema: schema{Type: "object", Properties: map[string]schema{
			"enterprise": {Type: "string"},
			"name":       {Type: "string"},
			"settings":   {Type: "object"},
		}, Required: []string{"enterprise", "settings"}}},
	}, true, []generatedParameter{{Name: "enterprise_id", In: "path", Scope: true, ScopeName: "enterprise"}}, nil)
	if !body.Required || !body.BodyOnly || len(body.Properties) != 0 {
		t.Fatalf("body metadata = %#v", body)
	}
	if len(body.AutoFill) != 1 || body.AutoFill[0].Name != "enterprise" || body.AutoFill[0].Parameter != "enterprise_id" {
		t.Fatalf("collision auto-fill = %#v", body.AutoFill)
	}
}

func TestExtractBodyRetainsCollisionAutoFillFormat(t *testing.T) {
	body := extractBody(map[string]requestMedia{
		"application/json": {Schema: schema{Type: "object", Properties: map[string]schema{
			"enterprise": {Type: "string", Format: "url"},
		}, Required: []string{"enterprise"}}},
	}, true, []generatedParameter{{Name: "enterprise_id", In: "path", Scope: true, ScopeName: "enterprise"}}, nil)
	if len(body.AutoFill) != 1 || body.AutoFill[0].Format != "url" {
		t.Fatalf("collision auto-fill = %#v", body.AutoFill)
	}
}

func TestExtractBodyRequiredEmptyObject(t *testing.T) {
	body := extractBody(map[string]requestMedia{"application/json": {Schema: schema{Type: "object"}}}, true, nil, nil)
	if !body.Required || !body.Empty {
		t.Fatalf("body metadata = %#v", body)
	}
}

func TestExtractBodyRequiredRootArrayIsNotEmptyObject(t *testing.T) {
	body := extractBody(map[string]requestMedia{"application/json": {Schema: schema{Type: "array"}}}, true, nil, nil)
	if !body.Required || body.Empty {
		t.Fatalf("body metadata = %#v", body)
	}
}

func TestExtractBodyComposedRequiredComplexPropertyIsBodyOnly(t *testing.T) {
	schemas := map[string]schema{
		"Create": {AllOf: []schema{
			{Ref: "#/components/schemas/Base"},
			{Type: "object", Properties: map[string]schema{"options": {Type: "array"}}, Required: []string{"options"}},
		}},
		"Base": {Type: "object", Properties: map[string]schema{
			"name":       {Type: "string"},
			"properties": {Type: "object"},
		}, Required: []string{"name", "properties"}},
	}
	body := extractBody(map[string]requestMedia{"application/json": {Schema: schema{Ref: "#/components/schemas/Create"}}}, true, nil, schemas)
	if !body.Required || !body.BodyOnly || len(body.Properties) != 0 {
		t.Fatalf("body metadata = %#v", body)
	}
}

func TestResolveCyclicSchemaReferenceStops(t *testing.T) {
	schemas := map[string]schema{
		"First":  {Ref: "#/components/schemas/Second"},
		"Second": {Ref: "#/components/schemas/First"},
	}
	resolved := resolve(schema{Ref: "#/components/schemas/First"}, schemas)
	if resolved.Ref == "" {
		t.Fatalf("resolved cyclic schema = %#v", resolved)
	}
}

func TestScopeParameterNamesIncludesAncestorsOnly(t *testing.T) {
	scopes := scopeParameterNames("/stages/{stage_id}/runs/{run_id}/commands/{command_id}", "run")
	if scopes["stage_id"] != "stage" || scopes["run_id"] != "pipeline-run" || scopes["command_id"] != "" {
		t.Fatalf("scopes = %#v", scopes)
	}
}

func TestAliasesUseCanonicalCommand(t *testing.T) {
	operations := []generatedOperation{
		{Generation: "v0", Noun: "geofence", Verb: "get", OperationID: "canonical"},
		{Generation: "v0", Noun: "the-geofence", Verb: "get", OperationID: "alias", AliasOf: "canonical"},
	}
	assignCommands(operations)
	if operations[1].Noun != "geofence" || len(operations[1].Command) != 2 || operations[1].Command[0] != "geofence" {
		t.Fatalf("alias command = %#v", operations[1])
	}
}

func TestPipelineCommandAndOperationAlwaysUseSideFamilyPrefix(t *testing.T) {
	operations := []generatedOperation{
		{Generation: "pipelines-v0", Noun: "command", Verb: "list", ScopeParent: "target-run"},
		{Generation: "pipelines-v0", Noun: "operation", Verb: "get", ScopeParent: "stage-run"},
	}
	assignCommands(operations)
	if operations[0].Noun != "pipeline-command" || operations[1].Noun != "pipeline-operation" {
		t.Fatalf("pipeline nouns = %#v", operations)
	}
}

func TestSuccessMediaReturnsEmptyForNoContent(t *testing.T) {
	if media := successMedia(map[string]response{"204": {}}); media != "" {
		t.Fatalf("successMedia() = %q, want empty", media)
	}
}

func TestValidateGenerationRejectsUnknownValue(t *testing.T) {
	if err := validateGeneration("v99"); err == nil {
		t.Fatal("validateGeneration() error = nil")
	}
}

func TestDeriveResponseEnvelopesByServiceFamily(t *testing.T) {
	operations := []generatedOperation{
		{Path: "/v2/devices/", Pagination: "apps-envelope"},
		{Path: "/v2/devices/{id}"},
		{Path: "/v2/scripts/{id}"},
		{Path: "/unwrapped/{id}", ResponseEnvelope: "explicit"},
	}
	deriveResponseEnvelopes(operations)
	if operations[0].ResponseEnvelope != "apps-envelope" || operations[1].ResponseEnvelope != "apps-envelope" {
		t.Fatalf("device envelopes = %#v", operations[:2])
	}
	if operations[2].ResponseEnvelope != "" || operations[3].ResponseEnvelope != "explicit" {
		t.Fatalf("unrelated envelopes = %#v", operations[2:])
	}
}

func TestServiceFamily(t *testing.T) {
	tests := map[string]string{
		"/v2/devices/{id}":         "v2/devices",
		"/v1/foundry/builds/{id}/": "v1/foundry",
		"/pipelines/v0/runs/":      "pipelines/v0",
	}
	for path, want := range tests {
		if got := serviceFamily(path); got != want {
			t.Fatalf("serviceFamily(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestExtractBodyIncludesNestedFieldsAndItemEnums(t *testing.T) {
	schemas := map[string]schema{
		"Filter": {Type: "object", Properties: map[string]schema{
			"platform": {Type: "array", Items: &schema{Type: "string", Enum: []any{"Android", "Apple"}}},
		}},
	}
	content := map[string]requestMedia{
		"application/json": {
			Schema: schema{
				Type: "object",
				Properties: map[string]schema{
					"report_type": {Type: "string", Enum: []any{"device"}},
					"filters":     {Ref: "#/components/schemas/Filter"},
				},
				Required: []string{"report_type"},
			},
		},
	}
	body := extractBody(content, true, nil, schemas)
	if len(body.Fields) != 2 {
		t.Fatalf("body fields = %#v", body.Fields)
	}
	if body.Fields[0].Path != "filters.platform" || len(body.Fields[0].Enum) != 2 || body.Fields[1].Path != "report_type" || !body.Fields[1].Required {
		t.Fatalf("body fields = %#v", body.Fields)
	}
}
