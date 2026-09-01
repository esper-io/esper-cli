package approval

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestShowRedactsRequestValuesAndApproveRejectsNonTerminal(t *testing.T) {
	t.Setenv(esperruntime.CredentialsFileEnvironment, filepath.Join(t.TempDir(), "creds.json"))
	store, err := esperruntime.NewApprovalStore()
	if err != nil {
		t.Fatal(err)
	}
	request, _, err := store.Request(esperruntime.ApprovalSpec{BaseURL: "https://example.test", Method: "POST", Path: "/things", Body: []byte(`{"token":"secret"}`)})
	if err != nil {
		t.Fatal(err)
	}
	command := NewCommand(&esperruntime.GlobalOptions{})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"show", request.ID})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "secret") || !strings.Contains(output.String(), "token") {
		t.Fatalf("show output = %q", output.String())
	}
	command = NewCommand(&esperruntime.GlobalOptions{})
	command.SetIn(strings.NewReader("approve " + request.ID + "\n"))
	command.SetArgs([]string{"approve", request.ID})
	if err := command.Execute(); err == nil || esperruntime.ExitCode(err) != 2 {
		t.Fatalf("non-terminal approve error = %v", err)
	}
}
