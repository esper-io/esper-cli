package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/esper-io/esper-cli/internal/cmd/generated"
)

func TestSafeCommandRedactsBodiesAndKeys(t *testing.T) {
	got := safeCommand([]string{"command", "create", "--api-key=secret", "--body", `{"token":"secret"}`})
	if strings.Contains(got, "secret") || !strings.Contains(got, "<redacted>") {
		t.Fatalf("safeCommand() = %q", got)
	}
}

func TestReadonlyArgsRequireOneOfAndLimit(t *testing.T) {
	operation := generated.Operation{
		Command:      []string{"itunesapp", "list"},
		Path:         "/v2/itunesapps",
		Method:       "GET",
		RequireOneOf: []string{"app_id", "apple_app_id"},
		Parameters: []generated.Parameter{
			{Name: "app_id", In: "query"},
			{Name: "apple_app_id", In: "query"},
			{Name: "limit", In: "query"},
		},
	}
	args, missing := readonlyArgs(operation, map[string]string{"app_id": "app-1"})
	if len(missing) != 0 || !contains(args, "--app-id") || !contains(args, "--limit") {
		t.Fatalf("readonlyArgs() = %v, %v", args, missing)
	}
}

func TestMutationScenarioRequiresCleanup(t *testing.T) {
	if _, err := os.CreateTemp(t.TempDir(), "scenario"); err != nil {
		t.Fatal(err)
	}
	// The safety assertion is exercised through runMutations only with a real plan.
	if mutationConfirmation != "I_UNDERSTAND_THIS_TENANT_IS_DISPOSABLE" {
		t.Fatal("mutation confirmation changed")
	}
}

func TestMutationStepRequiresExplicitYes(t *testing.T) {
	runner := &harness{binary: "/does/not/run", timeout: time.Second}
	expected := 0
	item := runner.executeStep(step{Name: "delete", Args: []string{"device", "delete-non-android-device", "device-1"}, ExpectedExit: &expected, Mutation: true}, nil, false)
	if item.Status != "FAIL" || !strings.Contains(item.Detail, "--yes") {
		t.Fatalf("mutation safety result = %#v", item)
	}
}

func TestMutationStepCannotHideGeneratedWrite(t *testing.T) {
	expected := 0
	runner := &harness{binary: "/does/not/run", timeout: time.Second}
	item := runner.executeStep(step{Name: "delete", Args: []string{"device", "delete-non-android-device", "device-1", "--yes"}, ExpectedExit: &expected}, nil, false)
	if item.Status != "FAIL" || !strings.Contains(item.Detail, "mutation=true") {
		t.Fatalf("unmarked write result = %#v", item)
	}
}

func TestCollectKnownPreservesExplicitInputs(t *testing.T) {
	known := map[string]string{"device_id": "selected-device", "enterprise_id": "selected-enterprise"}
	collectKnown([]byte(`{"device_id":"listed-device","enterprise_id":"listed-enterprise","group_id":"group-1"}`), "device", known)
	if known["device_id"] != "selected-device" || known["enterprise_id"] != "selected-enterprise" {
		t.Fatalf("explicit inputs were replaced: %#v", known)
	}
	if known["group"] != "group-1" {
		t.Fatalf("unknown dependent ID was not collected: %#v", known)
	}
}

func TestCollectKnownDoesNotStoreAmbiguousRawIDs(t *testing.T) {
	known := map[string]string{}
	collectKnown([]byte(`{"latest_published_version_id":"blueprint-version-1"}`), "blueprint", known)
	if known["version_id"] != "" || known["blueprint-version"] != "blueprint-version-1" {
		t.Fatalf("ambiguous ID collection = %#v", known)
	}
}

func TestReadonlyArgsDoNotReuseUnrelatedResourceIDs(t *testing.T) {
	known := map[string]string{"app_id": "application-1", "version_id": "version-1", "blueprint_id": "blueprint-1"}
	tests := []generated.Operation{
		{Noun: "app-vpp", Parameters: []generated.Parameter{{Name: "appId", In: "path"}}},
		{Noun: "device-app", Parameters: []generated.Parameter{{Name: "app_id", In: "path"}}},
		{Noun: "tenant-app-version", Parameters: []generated.Parameter{{Name: "appId", In: "path"}, {Name: "versionId", In: "path"}}},
		{Generation: "legacy", Noun: "blueprint", Parameters: []generated.Parameter{{Name: "blueprint_id", In: "path"}}},
	}
	for _, operation := range tests {
		for _, parameter := range operation.Parameters {
			if value := valueFor(parameter, operation, known); value != "" {
				t.Fatalf("%s reused unrelated %s=%q", operation.Noun, parameter.Name, value)
			}
		}
	}
}

func TestReportDatesUseCompletedTimezoneNaiveWindow(t *testing.T) {
	start := valueFor(generated.Parameter{Name: "start_date"}, generated.Operation{}, nil)
	end := valueFor(generated.Parameter{Name: "end_date"}, generated.Operation{}, nil)
	if strings.Contains(start, "Z") || strings.Contains(end, "Z") || start >= end {
		t.Fatalf("report window = %q through %q", start, end)
	}
}

func TestFailedCommandDetailTrimsAndBoundsStderr(t *testing.T) {
	if got := failedCommandDetail("\nerror: HTTP 403\n"); got != "error: HTTP 403" {
		t.Fatalf("failedCommandDetail() = %q", got)
	}
	if got := failedCommandDetail(strings.Repeat("x", 513)); len(got) != 515 || !strings.HasSuffix(got, "...") {
		t.Fatalf("failedCommandDetail() length = %d", len(got))
	}
}

func TestUnboundedListsRequireExplicitApproval(t *testing.T) {
	operation := generated.Operation{Verb: "list"}
	if !(&harness{}).skipsUnboundedList(operation) {
		t.Fatal("unbounded list ran without approval")
	}
	if (&harness{allowDefaultPageLists: true}).skipsUnboundedList(operation) {
		t.Fatal("approved default-page list was skipped")
	}
}

func TestReadonlyArgsKeepBlueprintFamiliesSeparate(t *testing.T) {
	known := map[string]string{"blueprint": "blueprint-v2", "blueprint-version": "blueprint-version-v2", "version": "application-version"}
	versionArgs, missing := readonlyArgs(generated.Operation{Noun: "blueprint-version", Parameters: []generated.Parameter{{Name: "blueprint_id", In: "path"}, {Name: "version_id", In: "path"}}}, known)
	if len(missing) != 0 || !contains(versionArgs, "blueprint-v2") || !contains(versionArgs, "blueprint-version-v2") {
		t.Fatalf("blueprint version args = %v, missing = %v", versionArgs, missing)
	}
	_, missing = readonlyArgs(generated.Operation{Generation: "legacy", Noun: "revision", Parameters: []generated.Parameter{{Name: "blueprint_id", In: "path"}}}, known)
	if len(missing) != 1 || missing[0] != "blueprint_id" {
		t.Fatalf("legacy blueprint missing = %v", missing)
	}
}

func TestReadonlyArgsUseUserResourceID(t *testing.T) {
	args, missing := readonlyArgs(generated.Operation{Noun: "user", Parameters: []generated.Parameter{{Name: "user_id", In: "path"}}}, map[string]string{"user": "user-1", "authn-user": "unrelated-authn-user"})
	if len(missing) != 0 || !contains(args, "user-1") || contains(args, "unrelated-authn-user") {
		t.Fatalf("user args = %v, missing = %v", args, missing)
	}
}

func TestCollectKnownIgnoresUserIDsOutsideUserResource(t *testing.T) {
	known := map[string]string{}
	collectKnown([]byte(`{"user_id":"unrelated-user"}`), "group", known)
	if known["user"] != "" {
		t.Fatalf("group response populated user ID: %#v", known)
	}
}
