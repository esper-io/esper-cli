package cmd

import "testing"

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
