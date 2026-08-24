package runtime

import (
	"encoding/json"
	"testing"
)

func TestPaginationUnwrappers(t *testing.T) {
	tests := []struct {
		name     string
		unwrap   func([]byte) (Page, error)
		input    string
		wantNext string
		wantPrev string
	}{
		{
			name:     "limit offset",
			unwrap:   UnwrapLimitOffset,
			input:    `{"limit":2,"offset":0,"count":3,"next":"/next","previous":null,"results":[{"id":"one"},{"id":"two"}]}`,
			wantNext: "/next",
		},
		{
			name:     "apps envelope",
			unwrap:   UnwrapAppsEnvelope,
			input:    `{"code":200,"message":"ok","content":{"count":3,"next":"/next","prev":"/previous","results":[{"id":"one"},{"id":"two"}]}}`,
			wantNext: "/next",
			wantPrev: "/previous",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, err := test.unwrap([]byte(test.input))
			if err != nil {
				t.Fatalf("unwrap() error = %v", err)
			}
			if page.Count != 3 || len(page.Results) != 2 || page.Next != test.wantNext || page.Previous != test.wantPrev {
				t.Fatalf("unwrap() = %#v", page)
			}
			var result map[string]string
			if err := json.Unmarshal(page.Results[0], &result); err != nil {
				t.Fatal(err)
			}
			if result["id"] != "one" {
				t.Fatalf("first result = %#v", result)
			}
		})
	}
}

func TestMarshalMergedResults(t *testing.T) {
	data, err := MarshalMergedResults([]json.RawMessage{json.RawMessage(`{"id":"one"}`), json.RawMessage(`{"id":"two"}`)})
	if err != nil {
		t.Fatalf("MarshalMergedResults() error = %v", err)
	}
	if got, want := string(data), `[{"id":"one"},{"id":"two"}]`; got != want {
		t.Fatalf("MarshalMergedResults() = %s, want %s", got, want)
	}
}
