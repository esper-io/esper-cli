# espercli

## Give your Agent a Helper Andi

`espercli` gives people and AI agents a practical command-line interface for
Esper. It turns the public Esper API surface into discoverable commands with
structured JSON output, stable exit codes, shell completion, and generated
agent guidance.

## What Is espercli?

Esper CLI is a Go client generated from Esper's canonical API specifications.
Run it directly from a terminal, use it in scripts, or let an agent discover
commands with `--help` before it acts.

The command tree is checked against the public API specification so command
names, flags, scopes, pagination, and destructive-operation metadata stay in
sync with the supported API surface.

## Why Your Agent Will Like It

- **Discoverable**: Every resource and operation has built-in help.
- **Structured**: `--json` preserves the raw API response for reliable parsing.
- **Scoped**: Commands use explicit resource scope flags such as `--enterprise`,
  `--device`, and `--pipeline`.
- **Guarded**: API writes require human approval, and destructive operations
  require a separate confirmation.
- **Current**: Newer API generations own the default command names; retained
  legacy routes are explicit rather than silent fallbacks.

## Try It

Create an Esper account at [esper.io/signup](https://www.esper.io/signup), then
configure the CLI with a tenant name, enterprise ID, and API key. Running
`configure` without flags prompts for each value.

```bash
espercli configure
espercli configure show
```

Ask the CLI what it can do, then fetch a small set of devices as JSON:

```bash
espercli device --help
espercli device list --limit 5 --json |
  jq '.content.results[] | {id, name, state}'
```

Set context when several commands should operate on the same resources:

```bash
espercli context set enterprise <enterprise-id>
espercli context set device <device-id>
espercli context get
```

The normal command form is:

```text
espercli <singular-noun> <verb> [positional IDs] [flags]
```

For example:

```bash
espercli device get <device-id>
espercli application upload <enterprise-id> --app-file ./app.apk
espercli geofence list --limit 20
```

## Install

### Release artifacts

GitHub releases contain archives for Linux, macOS, and Windows on amd64 and
arm64. Download the matching `espercli_<version>_<os>_<arch>` archive, extract
it, and place `espercli` on `PATH`.

CI snapshot builds are available from their GitHub Actions run as the
`espercli-snapshot` artifact. Snapshots are not published as releases.

### Homebrew

The release configuration generates an `espercli` formula for the
`esper-io/homebrew-tap` repository. After the tap is published:

```bash
brew tap esper-io/tap
brew install espercli
```

### Go install

Go 1.23 or newer is required.

```bash
go install github.com/esper-io/esper-cli/cmd/espercli@latest
```

## Safe Writes

Reads run normally. Before an API write, the CLI stops and prints an approval
ID without making the request:

```text
approval required for POST /v2/blueprints/
review: espercli approval show <id>
human approval: espercli approval approve <id>
```

A human reviews the sanitized request in an interactive terminal, then approves
that exact request. The original command must be retried unchanged. Approval is
bound to the method, target, query, body, and environment; it expires after 15
minutes and is consumed once.

Destructive operations require a second target confirmation after approval.
`--yes` skips only that confirmation. It never creates or bypasses approval.

## API Coverage

The live command tree is the canonical command reference:

```bash
espercli --help
espercli <resource> --help
espercli <resource> <operation> --help
```

Use `--json` for machine-readable responses. For supported paginated lists,
`--all --json` writes one merged result array.

Older API generations are available only where their behavior remains distinct:

```bash
espercli api legacy --help
espercli api v1 --help
```

Newer APIs own the standard command name. Legacy routes with a newer replacement
are removed instead of preserved as compatibility aliases.

## Agent Guidance

[`SKILL.md`](SKILL.md) is the generated, agent-agnostic Esper CLI guide. It
maps natural-language requests to the CLI, uses JSON for parsing, and follows
the same approval boundaries. It is generated from the same operation metadata
as the CLI and checked for drift in CI.

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

Run `espercli completion <shell> --help` for shell-specific persistent install
instructions.

## Development

The OpenAPI specifications and Esper overlay annotations drive code generation.
Do not edit generated command metadata by hand.

```bash
go run ./tools/codegen
go run ./tools/contractcheck
go test ./...
go vet ./...
go build ./...
```

The contract checker verifies that every supported public API operation remains
reachable and that flags, scopes, pagination, and destructive-operation metadata
match the specification. CI also checks deterministic code generation and
generated agent guidance.

## License

Apache-2.0. See [LICENSE](LICENSE).

## Tagline Credit

"Give your Agent a Helper Andi" is inspired by DataDog's
[Pup CLI](https://github.com/DataDog/pup) tagline, "Give Your Agent a Puppy."
