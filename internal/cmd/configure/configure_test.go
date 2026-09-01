package configure

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

func TestConfigureNonInteractiveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "creds.json")
	t.Setenv(esperruntime.CredentialsFileEnvironment, path)
	options := &esperruntime.GlobalOptions{Environment: "develop", APIKey: "secret-api-key"}
	command := &cobra.Command{}
	command.SetIn(strings.NewReader("enterprise-1\n"))
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	if err := runConfigure(command, options); err != nil {
		t.Fatalf("runConfigure() error = %v", err)
	}
	if stdout.String() != "Configuration saved.\n" {
		t.Fatalf("output = %q", stdout.String())
	}
	state, err := (&esperruntime.StateStore{Path: path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.Config.Environment != "develop" || state.Config.EnterpriseID != "enterprise-1" || state.Config.APIKey != "secret-api-key" {
		t.Fatalf("config = %#v", state.Config)
	}
}

func TestConfigurePromptsForMissingValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "creds.json")
	t.Setenv(esperruntime.CredentialsFileEnvironment, path)
	command := &cobra.Command{}
	command.SetIn(strings.NewReader("staging\nenterprise-1\nprompted-key\n"))
	var prompts bytes.Buffer
	command.SetErr(&prompts)
	command.SetOut(&bytes.Buffer{})
	if err := runConfigure(command, &esperruntime.GlobalOptions{}); err != nil {
		t.Fatalf("runConfigure() error = %v", err)
	}
	if prompts.String() != "Tenant name: Enterprise ID: API key: " {
		t.Fatalf("prompts = %q", prompts.String())
	}
	state, err := (&esperruntime.StateStore{Path: path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.Config.Environment != "staging" || state.Config.EnterpriseID != "enterprise-1" || state.Config.APIKey != "prompted-key" {
		t.Fatalf("config = %#v", state.Config)
	}
}

func TestConfigureShowRedactsAPIKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "creds.json")
	t.Setenv(esperruntime.CredentialsFileEnvironment, path)
	store := &esperruntime.StateStore{Path: path}
	if err := store.Save(esperruntime.State{Config: esperruntime.Config{Environment: "develop", EnterpriseID: "enterprise-1", APIKey: "secret-api-key"}}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		json    bool
		want    string
		notWant string
	}{
		{name: "human", want: "tenant_name: develop\nenterprise_id: enterprise-1\napi_key: **********-key\n", notWant: "secret-api-key"},
		{name: "json", json: true, want: `{"tenant_name":"develop","enterprise_id":"enterprise-1","api_key":"**********-key"}` + "\n", notWant: "secret-api-key"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := &cobra.Command{}
			var output bytes.Buffer
			command.SetOut(&output)
			if err := runShow(command, &esperruntime.GlobalOptions{JSON: test.json}); err != nil {
				t.Fatal(err)
			}
			if output.String() != test.want || strings.Contains(output.String(), test.notWant) {
				t.Fatalf("output = %q, want %q", output.String(), test.want)
			}
		})
	}
}

func TestRedactDoesNotExposeShortKeys(t *testing.T) {
	if got := redact("abc"); got != "***" {
		t.Fatalf("redact() = %q, want %q", got, "***")
	}
}
