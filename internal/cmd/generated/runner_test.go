package generated

import (
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
