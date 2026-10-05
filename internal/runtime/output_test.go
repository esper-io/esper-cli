package runtime

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	var output bytes.Buffer
	if err := WriteJSON(&output, []byte(`{"name":"kiosk","id":"one"}`)); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, `"name": "kiosk"`) || !strings.HasSuffix(got, "\n") {
		t.Fatalf("WriteJSON() = %q", got)
	}
}

func TestWriteHumanTable(t *testing.T) {
	var output bytes.Buffer
	if err := WriteHuman(&output, []byte(`[{"name":"kiosk","id":"one"}]`)); err != nil {
		t.Fatalf("WriteHuman() error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, "id") || !strings.Contains(got, "kiosk") {
		t.Fatalf("WriteHuman() = %q", got)
	}
}

func TestWriteHumanReadableResponses(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  []string
	}{
		{"paginated list", `{"count":2,"next":null,"results":[{"id":"one","name":"kiosk"},{"id":"two","name":"tablet"}]}`, []string{"count", "2", "results", "one", "kiosk", "two", "tablet"}},
		{"content list", `{"code":200,"content":[{"id":"one","name":"kiosk"}]}`, []string{"code", "200", "content", "kiosk"}},
		{"named list", `{"roles":[{"name":"Viewer"}],"total":1}`, []string{"roles", "Viewer", "total"}},
		{"nested detail", `{"id":"one","settings":{"enabled":true,"tags":["test","kiosk"]},"devices":[{"name":"tablet"}]}`, []string{"settings.enabled", "true", "settings.tags", "test, kiosk", "devices", "tablet"}},
		{"nested table cells", `[{"id":"one","settings":{"enabled":true},"tags":["test","kiosk"]},{"id":"two","settings":{"mode":"managed"}}]`, []string{"settings.enabled", "settings.mode", "true", "managed", "test, kiosk", "one", "two"}},
		{"scalar list", `["one","two",3]`, []string{"one", "two", "3"}},
		{"mixed list", `[{"id":"one"},"two",3]`, []string{"id=one", "two", "3"}},
		{"empty list", `{"results":[]}`, []string{"results", "No results."}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := WriteHuman(&output, []byte(test.input)); err != nil {
				t.Fatal(err)
			}
			for _, want := range test.want {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("output = %q, missing %q", output.String(), want)
				}
			}
			if strings.ContainsAny(output.String(), "{}[]") {
				t.Fatalf("human output contains JSON structure: %q", output.String())
			}
		})
	}
}

func TestWriteHumanTableKeepsCellsOnOneLine(t *testing.T) {
	var output bytes.Buffer
	if err := WriteHuman(&output, []byte(`[{"name":"first\nsecond\tthird"}]`)); err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(output.String()), "\n"); len(lines) != 2 {
		t.Fatalf("table cells broke alignment: %q", output.String())
	}
}

func TestWriteHumanWideTableShowsSummary(t *testing.T) {
	var output bytes.Buffer
	if err := WriteHuman(&output, []byte(`[{"id":"one","name":"kiosk","state":"online","platform":"ANDROID","last_seen":"today","extra_a":1,"extra_b":2,"extra_c":3,"extra_d":4,"extra_e":5}]`)); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"one", "kiosk", "online", "ANDROID", "today", "Showing 8 of 10 fields. Use --json"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("summary output = %q, missing %q", output.String(), want)
		}
	}
}

func TestWriteJSONPreservesResponseEnvelope(t *testing.T) {
	var output bytes.Buffer
	if err := WriteJSON(&output, []byte(`{"count":1,"results":[{"settings":{"enabled":true}}]}`)); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"count": 1`, `"results": [`, `"settings": {`, `"enabled": true`} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("JSON output changed shape: %q", output.String())
		}
	}
}

func TestUnwrapResponseEnvelope(t *testing.T) {
	input := []byte(`{"code":200,"message":"ok","content":{"id":"device-1"}}`)
	got, err := UnwrapResponseEnvelope(input, "apps-envelope")
	if err != nil || string(got) != `{"id":"device-1"}` {
		t.Fatalf("UnwrapResponseEnvelope() = %s, %v", got, err)
	}
	raw, err := UnwrapResponseEnvelope(input, "")
	if err != nil || !bytes.Equal(raw, input) {
		t.Fatalf("raw response = %s, %v", raw, err)
	}
}

func TestConfirm(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		yes    bool
		wanted bool
	}{
		{name: "accepted", input: "yes\n", wanted: true},
		{name: "declined", input: "no\n", wanted: false},
		{name: "bypassed", yes: true, wanted: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			got, err := Confirm(strings.NewReader(test.input), &output, "device one", 1, test.yes)
			if err != nil {
				t.Fatalf("Confirm() error = %v", err)
			}
			if got != test.wanted {
				t.Fatalf("Confirm() = %v, want %v", got, test.wanted)
			}
			if !test.yes && !strings.Contains(output.String(), "device one (1 target(s))") {
				t.Fatalf("prompt = %q", output.String())
			}
		})
	}
}
