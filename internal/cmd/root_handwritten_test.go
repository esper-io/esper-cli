package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/esper-io/esper-cli/internal/version"
)

func TestSecureADBCommandRegistered(t *testing.T) {
	command, _, err := NewRootCommand().Find([]string{"secureadb", "connect"})
	if err != nil {
		t.Fatalf("find secureadb connect: %v", err)
	}
	if command.CommandPath() != "espercli secureadb connect" {
		t.Fatalf("command path = %q", command.CommandPath())
	}
	if command.Flags().Lookup("device") == nil {
		t.Fatal("secureadb connect has no --device flag")
	}
}

func TestStateCommandsRegistered(t *testing.T) {
	for _, path := range [][]string{{"configure"}, {"configure", "show"}, {"context", "set"}, {"context", "get"}, {"context", "clear"}} {
		command, _, err := NewRootCommand().Find(path)
		if err != nil || command.CommandPath() != "espercli "+strings.Join(path, " ") {
			t.Fatalf("find %v = %q, %v", path, command.CommandPath(), err)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	originalVersion, originalCommit, originalDate := version.Version, version.Commit, version.Date
	t.Cleanup(func() {
		version.Version, version.Commit, version.Date = originalVersion, originalCommit, originalDate
	})
	version.Version, version.Commit, version.Date = "2.1.0", "abc123", "2026-08-25T12:00:00Z"

	tests := []struct {
		name      string
		arguments []string
		want      string
	}{
		{name: "human", arguments: []string{"version"}, want: "espercli 2.1.0 (commit abc123, built 2026-08-25T12:00:00Z)\n"},
		{name: "json", arguments: []string{"version", "--json"}, want: `{"version":"2.1.0","commit":"abc123","date":"2026-08-25T12:00:00Z"}` + "\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := NewRootCommand()
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&output)
			command.SetArgs(test.arguments)
			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if output.String() != test.want {
				t.Fatalf("output = %q, want %q", output.String(), test.want)
			}
		})
	}

	command := NewRootCommand()
	versionCommand, _, err := command.Find([]string{"version"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(versionCommand.CommandPath(), "espercli version") || versionCommand.Commands()[0].Name() != "list" {
		t.Fatal("version API subcommands were not preserved")
	}
}

func TestCompletionCommandsWriteScriptsToStdout(t *testing.T) {
	tests := []struct {
		shell  string
		marker string
	}{
		{shell: "bash", marker: "bash completion V2"},
		{shell: "zsh", marker: "#compdef espercli"},
		{shell: "fish", marker: "complete -c espercli"},
		{shell: "powershell", marker: "Register-ArgumentCompleter"},
	}
	for _, test := range tests {
		t.Run(test.shell, func(t *testing.T) {
			command := NewRootCommand()
			var stdout, stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			command.SetArgs([]string{"completion", test.shell})
			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !strings.Contains(stdout.String(), test.marker) {
				t.Fatalf("completion output does not contain %q", test.marker)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

func TestCompletionHelpIncludesInstallInstructions(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			command := NewRootCommand()
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&output)
			command.SetArgs([]string{"completion", shell, "--help"})
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "espercli completion "+shell) {
				t.Fatalf("help does not contain install instruction:\n%s", output.String())
			}
		})
	}
}
