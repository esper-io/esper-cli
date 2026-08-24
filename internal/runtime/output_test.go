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
