package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/esper-io/esper-cli/internal/cmd/generated"
)

func TestRenderSkillDeterministic(t *testing.T) {
	first := renderSkill(generated.Operations())
	second := renderSkill(generated.Operations())
	if !bytes.Equal(first, second) {
		t.Fatal("renderSkill() changed between identical inputs")
	}
}

func TestCommittedSkillCurrent(t *testing.T) {
	want := renderSkill(generated.Operations())
	path := filepath.Join("..", "..", defaultOutputPath)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; run go run ./tools/skillgen", path)
	}
}

func TestRenderSkillIncludesGeneratedAndHandWrittenCommands(t *testing.T) {
	content := string(renderSkill([]generated.Operation{
		{Command: []string{"device", "list"}, Summary: "List devices"},
		{Command: []string{"device", "delete"}, Summary: "Delete device", Destructive: true},
		{Command: []string{"device", "list"}, Summary: "Scoped list devices"},
		{Command: []string{"old", "alias"}, AliasOf: "canonical"},
	}))
	for _, expected := range []string{
		"`espercli secureadb connect --device <id>`",
		"`espercli version`",
		"`espercli device list` - List devices",
		"`espercli device delete` - Delete device **destructive**",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("generated skill does not contain %q", expected)
		}
	}
	if strings.Contains(content, "espercli old alias") {
		t.Fatal("generated skill includes an alias-only operation")
	}
}
