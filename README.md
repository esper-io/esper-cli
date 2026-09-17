# espercli

`espercli` is a Go command-line client for the public Esper APIs. Its command
tree is generated from the canonical API specifications and checked for complete
public-operation coverage. The CLI is designed for direct use and for automation
through stable JSON output, exit codes, help, completion, and the `/esper` agent
skill.

## Install

### Release artifacts

GoReleaser builds archives for Linux, macOS, and Windows on amd64 and arm64.
Download the matching `espercli_<version>_<os>_<arch>` archive from the GitHub
release, extract it, and place `espercli` on `PATH`.

CI snapshot builds are attached to their GitHub Actions run as the
`espercli-snapshot` artifact. Snapshots are not published as releases.

### Homebrew tap placeholder

The release configuration generates an `espercli` formula for the
`esper-io/homebrew-tap` repository. After tap publishing is enabled:

```bash
brew tap esper-io/tap
brew install espercli
```

### Go install

Go 1.23 or newer is required. After the Go rewrite is published as a release:

```bash
go install github.com/esper-io/esper-cli/cmd/espercli@latest
```

## Quick Start

Configure a tenant name, enterprise ID, and API key. Without flags, `configure`
prompts for all three values.

```bash
espercli configure
espercli configure show
```

Set active resource IDs when commands should use context fallback:

```bash
espercli context set enterprise <enterprise-id>
espercli context set device <device-id>
espercli context get
```

List devices and process the raw API envelope:

```bash
espercli device list --limit 5 --json |
  jq '.content.results[] | {id, name, state}'
```

Use `--help` to inspect any group or operation:

```bash
espercli device --help
espercli device get --help
```

## Command Grammar

The standard form is:

```text
espercli <singular-noun> <verb> [positional IDs] [flags]
```

Examples:

```bash
espercli device list --limit 20
espercli device get <device-id>
espercli application upload <enterprise-id> --app-file ./app.apk
```

Key rules:

- The newest API generation owns the bare noun/verb command. Older collisions
  remain available under `espercli api <generation> <noun> <verb>`, such as
  `espercli api legacy device list`.
- Parent-scoped routes use scope flags such as `--enterprise`, `--device`, or
  `--pipeline`. Required device, app, group, and enterprise IDs can fall back to
  active context where the canonical parameter name supports it.
- JSON request bodies accept scalar property flags or `--body`. `--body` accepts
  inline JSON, `@path`, or `-` for stdin and cannot be mixed with property flags.
- Every API write requires a one-time human approval. A write first prints an
  approval ID; a human must review it with `espercli approval show <id>` and run
  `espercli approval approve <id>` from an interactive terminal before the exact
  request can be retried. Approval binds the method, target, query, body, and
  environment, expires after 15 minutes, and is consumed once.
- Destructive operations also prompt for the exact target and count after human
  approval. `--yes` skips only that second prompt; it never bypasses approval.
- `--json` writes the raw API JSON envelope without field-name or shape changes.
  `--all --json` writes one merged result array for supported paginated lists.

Exit codes are stable:

| Code | Meaning |
|---:|---|
| `0` | Success |
| `1` | API or user error |
| `2` | Usage error or cancelled confirmation |
| `3` | Authentication or configuration error |
| `4` | Network or timeout error |

## Human Approval

For example, a create command from an agent or script exits before making an API
call and prints an approval ID:

```text
approval required for POST /v2/blueprints/
review: espercli approval show <id>
human approval: espercli approval approve <id>
```

The human terminal command displays only a sanitized request summary. Request
bodies, tokens, passwords, and API keys are not stored in the approval ledger.
The original command must then be retried unchanged.

## Shell Completion

Completion scripts are written to stdout. The CLI does not edit shell startup
files.

```bash
source <(espercli completion bash)
source <(espercli completion zsh)
espercli completion fish | source
```

```powershell
espercli completion powershell | Out-String | Invoke-Expression
```

Run `espercli completion <shell> --help` for the shell-specific persistent
installation command.

## `/esper` Skill

`.claude/commands/esper.md` provides the `/esper` Claude Code command. It maps
natural-language requests to the generated command tree, uses JSON for parsing,
and requires confirmation before destructive actions.

The skill is generated from the same operation metadata as the CLI and checked
for drift in CI. To use it in every project, install the file in the user-level
Claude Code commands directory.

## Development

The canonical OpenAPI specifications and Esper overlay annotations drive Go
command generation. Generated command metadata must not be edited by hand.

```bash
go run ./tools/codegen
go run ./tools/contractcheck
go test ./...
go vet ./...
go build ./...
```

The contract checker verifies that every public API operation remains reachable
and that flags, scopes, pagination, and destructive-operation metadata match the
specification. Tests use committed HTTP fixtures and must not require live
credentials. CI also checks deterministic code generation and the generated
`/esper` skill.

The repeatable command harness is documented in
[`docs/command-harness.md`](docs/command-harness.md):

```bash
./scripts/test-commands.sh offline
./scripts/test-commands.sh live-readonly
./scripts/test-commands.sh live-mutations
```

Offline mode checks every generated command and hand-written command through
`--help`. Live read-only mode runs bounded GET operations. Mutation mode requires
an explicit disposable-enterprise confirmation and a scenario cleanup section.
All modes write machine-readable reports under `dist/`.

The previous Python CLI is preserved on the `python-legacy` branch. New Go CLI
work belongs on `go-rewrite` until that branch becomes the default.

## License

Apache-2.0. See [LICENSE](LICENSE).
