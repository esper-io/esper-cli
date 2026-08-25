package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/esper-io/esper-cli/internal/cmd/generated"
)

const defaultOutputPath = ".claude/commands/esper.md"

type commandDoc struct {
	Path        string
	Summary     string
	Destructive bool
}

func main() {
	check := flag.Bool("check", false, "fail if the committed skill differs from generated content")
	output := flag.String("output", defaultOutputPath, "skill output path")
	flag.Parse()

	content := renderSkill(generated.Operations())
	if *check {
		current, err := os.ReadFile(*output)
		if err != nil {
			fatalf("read %s: %v", *output, err)
		}
		if !bytes.Equal(current, content) {
			fatalf("%s is stale; run go run ./tools/skillgen", *output)
		}
		return
	}
	if err := os.WriteFile(*output, content, 0o644); err != nil {
		fatalf("write %s: %v", *output, err)
	}
}

func renderSkill(operations []generated.Operation) []byte {
	commandsByPath := make(map[string]commandDoc)
	for _, operation := range operations {
		if operation.AliasOf != "" {
			continue
		}
		path := strings.Join(operation.Command, " ")
		doc, exists := commandsByPath[path]
		if !exists {
			doc = commandDoc{Path: path, Summary: oneLine(operation.Summary)}
		}
		doc.Destructive = doc.Destructive || operation.Destructive
		commandsByPath[path] = doc
	}
	paths := make([]string, 0, len(commandsByPath))
	for path := range commandsByPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var output strings.Builder
	output.WriteString("---\n")
	output.WriteString("description: Manage Esper resources through the spec-generated espercli command tree. Accepts natural-language requests.\n")
	output.WriteString("argument-hint: '[what you want to do, for example: list inactive devices]'\n")
	output.WriteString("allowed-tools: Bash\n")
	output.WriteString("---\n\n")
	output.WriteString("You are an Esper fleet management assistant. Translate the user's request into the smallest safe set of `espercli` commands.\n\n")
	output.WriteString("**User request:** $ARGUMENTS\n\n")
	output.WriteString("## Operating Rules\n\n")
	output.WriteString("1. Run `espercli <command> --help` before execution when required arguments or flags are not explicit below. Never invent flags.\n")
	output.WriteString("2. Use `--json` when parsing output. Keep stdout machine-readable and use exit codes to detect failure.\n")
	output.WriteString("3. Ask for explicit confirmation before destructive operations. Add `--yes` only after the user confirms.\n")
	output.WriteString("4. Prefer `--all` only when the user asks for complete paginated results.\n")
	output.WriteString("5. Use `--environment` and `--api-key` only when the user explicitly supplies overrides; otherwise rely on the configured environment.\n")
	output.WriteString("6. Do not call an API operation merely to discover whether it is safe. Use help and the command reference.\n\n")
	output.WriteString("## Hand-Written Commands\n\n")
	output.WriteString("- `espercli secureadb connect --device <id>` - Open a pinned mutual-TLS ADB relay.\n")
	output.WriteString("- `espercli completion <bash|fish|powershell|zsh>` - Write a shell completion script to stdout.\n")
	output.WriteString("- `espercli version` - Show build version, commit, and date.\n\n")
	output.WriteString("## Spec-Generated Command Tree\n\n")
	output.WriteString("Commands marked **destructive** require confirmation or `--yes`. Use each command's `--help` for positional arguments, request-body flags, scope flags, and pagination options.\n")

	currentGroup := ""
	for _, path := range paths {
		doc := commandsByPath[path]
		group := strings.Split(path, " ")[0]
		if group != currentGroup {
			currentGroup = group
			output.WriteString("\n### ")
			output.WriteString(group)
			output.WriteString("\n\n")
		}
		output.WriteString("- `espercli ")
		output.WriteString(doc.Path)
		output.WriteString("` - ")
		if doc.Summary == "" {
			output.WriteString("API operation")
		} else {
			output.WriteString(doc.Summary)
		}
		if doc.Destructive {
			output.WriteString(" **destructive**")
		}
		output.WriteString("\n")
	}
	return []byte(output.String())
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func fatalf(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, "skillgen: "+format+"\n", arguments...)
	os.Exit(1)
}
