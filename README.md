# espercli

## Give your Agent a Helper Andi

`espercli` is a command-line interface (CLI) for Esper. People and AI agents can
use it to send requests to the public Esper API.

## What Is espercli?

Esper CLI uses Go. Automation generates its commands from the public Esper API
specifications. Use the CLI in a terminal or a script. Use `--help` to examine a
command before you run it.

Automated checks compare the commands with the API specifications. The checks
include command names, flags, resource IDs, and list pages.

## Features

- Use `--help` to see the available commands and flags.
- Use `--json` to get the full API response as JSON.
- Use flags such as `--enterprise`, `--device`, and `--pipeline` to select resources.
- Get human approval before a command changes data through the API.
- Give a second confirmation before an operation that can delete data or reset a device.
- Use shell completion to enter commands.
- Use exit codes to check the result in a script.

## Try It

Create an Esper account at [esper.io/signup](https://www.esper.io/signup).
Run `configure` without flags. Enter the tenant name, enterprise ID, and API key
when the CLI asks for them.

```bash
espercli configure
espercli configure show
```

Show the device commands:

```bash
espercli device --help
```

Get up to five devices as JSON. Use `jq` to select the fields:

```bash
espercli device list --limit 5 --json |
  jq '.content.results[] | {id, name, state}'
```

Set the active enterprise and device to use them in later commands:

```bash
espercli context set enterprise <enterprise-id>
espercli context set device <device-id>
espercli context get
```

Use this command format:

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

This branch contains only the Go CLI. Install a release file or use `go install`.
Do not use `pip install .`.

### Release Files

GitHub releases contain archives for Linux, macOS, and Windows on amd64 and arm64.

1. Download the `espercli_<version>_<os>_<arch>` archive for your computer.
2. Extract the archive.
3. Put `espercli` in a directory on `PATH`.

For a test build, download the `espercli-snapshot` artifact from a GitHub Actions
run. Test builds are not releases.

### Homebrew

Release automation generates a Homebrew formula for `esper-io/homebrew-tap`.
The tap is not yet published. After publication, use these commands:

```bash
brew tap esper-io/tap
brew install espercli
```

### Go install

Install Go 1.23 or a later version.

```bash
go install github.com/esper-io/esper-cli/cmd/espercli@latest
```

## Safe Writes

Read commands do not need approval. Before an API write, the CLI stops and shows
an approval ID. It does not send the write request:

```text
approval required for POST /v2/blueprints/
review: espercli approval show <id>
human approval: espercli approval approve <id>
```

1. Run `espercli approval show <id>` to examine the request. The CLI hides secret values.
2. Run `espercli approval approve <id>` in an interactive terminal. A human must give the approval.
3. Run the original command again without changes.

Approval applies only to that request method, target, query, body, and environment.
Approval expires after 15 minutes. You can use it only once.

An operation that can delete data or reset a device needs a second target confirmation.
Use `--yes` to omit only this second confirmation. It does not give human approval
or remove the approval requirement.

## API Coverage

Use the installed CLI to see the available commands:

```bash
espercli --help
espercli <resource> --help
espercli <resource> <operation> --help
```

Use `--json` to read responses in scripts. For lists that support pages, use
`--all --json` to combine all pages into one result array.

Some commands use older APIs because they have functions that newer APIs do not
replace. To see these commands, use:

```bash
espercli api legacy --help
espercli api v1 --help
```

When a newer API replaces an older API, the standard command uses the newer API.
The CLI removes the old command instead of keeping an alias.

## Secure ADB Host Authorization

`espercli secureadb connect` sends the local ADB public key in the session request.
Supported Esper Foundation devices can use this key to approve the ADB host.
The CLI selects the key in this order:

1. If `ESPER_ADB_PUB_KEY` is set, use the public-key file at that path.
2. Otherwise, examine the private-key paths in `ADB_VENDOR_KEYS` in order. Use
   the first available matching `.pub` file.
3. If no matching file is available, use `~/.android/adbkey.pub`.

If the key file is missing, the CLI tries `adb start-server` once. This step has
a 30-second timeout. If the key is still missing, the CLI gives a warning. It
continues without sending a host public key.

If the key file is empty or cannot be read, the CLI stops before it creates a
session. Human approval includes the selected public key. If you change the key,
get new approval.

## Agent Guidance

[`SKILL.md`](SKILL.md) gives AI agents instructions for the CLI. It explains how
to select commands, read JSON, and get human approval. Automation generates this
file from the same command data as the CLI. Automated checks find differences
between the guide and the command data.

## Shell Completion

The CLI writes completion scripts to standard output. It does not change shell
startup files.

```bash
source <(espercli completion bash)
source <(espercli completion zsh)
espercli completion fish | source
```

```powershell
espercli completion powershell | Out-String | Invoke-Expression
```

Run `espercli completion <shell> --help` for installation instructions for your
shell. These instructions explain how to keep completion after you close the shell.

## Development

Automation generates commands from OpenAPI specifications and Esper annotations.
Do not manually change the generated command data.

```bash
go run ./tools/codegen
go run ./tools/contractcheck
go test ./...
go vet ./...
go build ./...
```

The contract checker makes sure each supported public API operation has a command.
It also checks flags, resource scopes, list pages, and confirmation requirements
against the specifications. Automated checks make sure generation gives the same
result for the same input. They also check the agent guide.

## License

Apache-2.0. See [LICENSE](LICENSE).

## Tagline Credit

"Give your Agent a Helper Andi" is inspired by DataDog's
[Pup CLI](https://github.com/DataDog/pup) tagline, "Give Your Agent a Puppy."
