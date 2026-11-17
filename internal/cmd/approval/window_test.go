package approval

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestWindowConfirmation(t *testing.T) {
	for _, test := range []struct {
		input io.Reader
		valid bool
	}{
		{strings.NewReader("start unrestricted window\n"), true},
		{strings.NewReader("yes\n"), false},
		{strings.NewReader("start unrestricted window"), false},
		{iotest.ErrReader(io.ErrUnexpectedEOF), false},
	} {
		var stderr bytes.Buffer
		err := confirmWindow(test.input, &stderr)
		if (err == nil) != test.valid {
			t.Fatalf("confirmation error = %v, valid = %v", err, test.valid)
		}
		for _, warning := range []string{"15 minutes", "ALL tenants", "destructive", "Secure ADB", `"start unrestricted window"`} {
			if !strings.Contains(stderr.String(), warning) {
				t.Fatalf("warning missing %q: %s", warning, stderr.String())
			}
		}
	}
}

func TestWindowStatusAndStop(t *testing.T) {
	t.Setenv(esperruntime.CredentialsFileEnvironment, filepath.Join(t.TempDir(), "creds.json"))
	store, _ := esperruntime.NewApprovalStore()
	started, err := store.StartWindow()
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store.Path)
	for _, action := range []string{"status", "status", "stop", "stop"} {
		command := NewCommand(&esperruntime.GlobalOptions{JSON: true})
		var stdout, stderr bytes.Buffer
		command.SetOut(&stdout)
		command.SetErr(&stderr)
		command.SetIn(iotest.ErrReader(io.ErrUnexpectedEOF))
		command.SetArgs([]string{"window", action})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		var status esperruntime.ApprovalWindowStatus
		if err := json.Unmarshal(stdout.Bytes(), &status); err != nil || stderr.Len() != 0 || status.Active != (action == "status") {
			t.Fatalf("%s output = %s, stderr = %s, error = %v", action, stdout.String(), stderr.String(), err)
		}
		if action == "status" {
			after, _ := os.ReadFile(store.Path)
			if !bytes.Equal(before, after) || *status.Window != *started.Window {
				t.Fatal("status changed window")
			}
		}
	}
	var human bytes.Buffer
	if err := writeWindow(&human, false, started); err != nil || !strings.Contains(human.String(), "active\n  Started:") || !strings.Contains(human.String(), "Expires:") {
		t.Fatalf("human window output = %q, %v", human.String(), err)
	}
}
