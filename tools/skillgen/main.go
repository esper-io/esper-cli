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

const defaultOutputPath = "SKILL.md"

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
	output.WriteString("name: espercli\n")
	output.WriteString("description: Esper API CLI with spec-generated commands for fleet, application, and tenant management.\n")
	output.WriteString("metadata:\n")
	output.WriteString("  repository: https://github.com/esper-io/esper-cli\n")
	output.WriteString("  tags: esper,cli,fleet-management,devices,applications\n")
	output.WriteString("---\n\n")
	output.WriteString("# Esper CLI\n\n")
	output.WriteString("Use `espercli` to manage Esper resources through the spec-generated command tree. Inspect command help before acting and choose the smallest safe set of commands.\n\n")
	output.WriteString("## Quick Start\n\n")
	output.WriteString("Create an Esper account at [esper.io/signup](https://www.esper.io/signup), then configure the CLI with your tenant name, enterprise ID, and API key.\n\n")
	output.WriteString("```bash\n")
	output.WriteString("espercli configure\n")
	output.WriteString("espercli device list --limit 5 --json\n")
	output.WriteString("```\n\n")
	output.WriteString("## Operating Rules\n\n")
	output.WriteString("1. Run `espercli <command> --help` before execution when required arguments or flags are not explicit below. Never invent flags.\n")
	output.WriteString("2. Use `--json` when parsing output. Keep stdout machine-readable and use exit codes to detect failure.\n")
	output.WriteString("3. For every API write, wait for the user to run `espercli approval approve <id>` from their terminal. Never run approval approve yourself; `--yes` does not bypass human approval.\n")
	output.WriteString("4. Prefer `--all` only when the user asks for complete paginated results.\n")
	output.WriteString("5. Use `--environment` and `--api-key` only when the user explicitly supplies overrides; otherwise rely on the configured environment.\n")
	output.WriteString("6. Do not call an API operation merely to discover whether it is safe. Use help and the command reference.\n\n")
	output.WriteString("7. For Linux, iOS, and Windows devices, use current `device` commands and `device-app list`. Do not fall back to `api legacy app list`, `api legacy status get`, or `device-app get` when a current device API returns an error; those legacy routes are Android compatibility surfaces.\n\n")
	output.WriteString("## Hand-Written Commands\n\n")
	output.WriteString("- `espercli configure [--environment <name>] [--api-key <key>]` - Store Esper credentials.\n")
	output.WriteString("- `espercli configure show` - Show the environment and redacted API key.\n")
	output.WriteString("- `espercli context set <device|app|group|enterprise> <id>` - Set active context.\n")
	output.WriteString("- `espercli context get [device|app|group|enterprise]` - Show active context.\n")
	output.WriteString("- `espercli context clear <device|app|group|enterprise>|--all` - Clear active context.\n")
	output.WriteString("- `espercli approval show <id>` - Show a sanitized pending API write.\n")
	output.WriteString("- `espercli approval approve <id>` - Human-only terminal approval for one pending API write.\n")
	output.WriteString("- `espercli discover <query-or-docs-url>` - Find commands by name, request fields, OpenAPI metadata, or an api.esper.io documentation URL.\n")
	output.WriteString("- `espercli secureadb connect --device <id>` - Open a pinned mutual-TLS ADB relay.\n")
	output.WriteString("- `espercli completion <bash|fish|powershell|zsh>` - Write a shell completion script to stdout.\n")
	output.WriteString("- `espercli version` - Show build version, commit, and date.\n\n")
	output.WriteString("## Spec-Generated Operations\n\n")
	output.WriteString("All API writes require a one-time human approval. Commands marked **destructive** also require confirmation unless `--yes` is supplied after approval. Use each command's `--help` for positional arguments, request-body flags, nested body fields, scope flags, and pagination options. `espercli api <generation> ...` accesses older colliding API generations; it is not a general API escape hatch.\n")

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
