package context

import (
	"bytes"
	"path/filepath"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestContextSetGetAndClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "creds.json")
	t.Setenv(esperruntime.CredentialsFileEnvironment, path)
	options := &esperruntime.GlobalOptions{}

	execute := func(arguments ...string) string {
		t.Helper()
		command := NewCommand(options)
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetErr(&output)
		command.SetArgs(arguments)
		if err := command.Execute(); err != nil {
			t.Fatalf("Execute(%v) error = %v", arguments, err)
		}
		return output.String()
	}

	if got := execute("set", "device", "device-1"); got != "device: device-1\n" {
		t.Fatalf("set output = %q", got)
	}
	if got := execute("set", "enterprise", "enterprise-1"); got != "enterprise: enterprise-1\n" {
		t.Fatalf("set output = %q", got)
	}
	if got := execute("get"); got != "device: device-1\napp: <unset>\ngroup: <unset>\nenterprise: enterprise-1\n" {
		t.Fatalf("get output = %q", got)
	}
	if got := execute("clear", "device"); got != "device: <unset>\n" {
		t.Fatalf("clear output = %q", got)
	}
	if got := execute("clear", "--all"); got != "device: <unset>\napp: <unset>\ngroup: <unset>\nenterprise: <unset>\n" {
		t.Fatalf("clear all output = %q", got)
	}

	state, err := (&esperruntime.StateStore{Path: path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.Active != (esperruntime.ActiveContext{}) {
		t.Fatalf("active context = %#v", state.Active)
	}
}

func TestContextRejectsInvalidResourceAndClearShape(t *testing.T) {
	t.Setenv(esperruntime.CredentialsFileEnvironment, filepath.Join(t.TempDir(), "creds.json"))
	tests := [][]string{
		{"set", "pipeline", "pipeline-1"},
		{"clear"},
		{"clear", "device", "--all"},
	}
	for _, arguments := range tests {
		command := NewCommand(&esperruntime.GlobalOptions{})
		command.SetArgs(arguments)
		if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
			t.Fatalf("Execute(%v) error = %v", arguments, err)
		}
	}
}
