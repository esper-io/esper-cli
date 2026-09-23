package version

import "testing"

func TestInfoString(t *testing.T) {
	info := Info{Version: "2.1.0", Commit: "abc123", Date: "2026-08-25T12:00:00Z"}
	want := "espercli 2.1.0 (commit abc123, built 2026-08-25T12:00:00Z)"
	if got := info.String(); got != want {
		t.Fatalf("Info.String() = %q, want %q", got, want)
	}
}
