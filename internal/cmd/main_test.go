package cmd

import (
	"os"
	"path/filepath"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "espercli-command-tests-")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv(esperruntime.CredentialsFileEnvironment, filepath.Join(directory, "creds.json"))
	_ = os.Unsetenv(esperruntime.EnvironmentVariable)
	_ = os.Unsetenv(esperruntime.APIKeyVariable)
	code := m.Run()
	_ = os.RemoveAll(directory)
	os.Exit(code)
}
