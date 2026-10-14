<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/esper-logo-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/esper-logo-light.svg">
    <img src="assets/esper-logo-light.svg" alt="Esper" width="252" height="80">
  </picture>
</p>

# Esper CLI 2.0

## Give your Agent a Helper Andi

`espercli` is a command-line interface (CLI) for Esper. People and AI agents can
use it to send requests to the public Esper API.

## What Is espercli?

Esper CLI is an AI-agent-ready command-line interface for device management.
It gives people and agents access to Esper APIs for devices, apps, blueprints,
and pipelines. Your agents can check device state, manage apps, run operations,
and find issues without working out each API request.

AI agents need clear commands and results to manage devices. Esper CLI lets
agents find the right command, read results as JSON, and use them in workflows.
Built-in help explains each command. Human approval keeps you in control of
changes to your devices and data.

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

## Migration from the Python CLI

This Go CLI is a brand-new implementation. It is not an update from the Python CLI.
There is no migration path from the Python CLI to this version. Do not assume
that existing commands or scripts work without changes.

The Python CLI is not abandoned. You can continue to use the
[Python version on `develop`](https://github.com/esper-io/esper-cli/tree/develop).
You do not need to switch to the Go CLI.

## Install

Release files are available for Linux, macOS, and Windows on amd64 and arm64.

### macOS

After the first tagged release, install with Homebrew:

```bash
brew install esper-io/tap/espercli
```

To update an existing installation:

```bash
brew update
brew upgrade espercli
```

You can also download a macOS release archive from GitHub releases. Extract the
archive. Put `espercli` in a directory on `PATH`.

### Linux

```bash
curl -fsSL https://raw.githubusercontent.com/esper-io/esper-cli/main/install.sh | sh
```

The script detects amd64 or arm64, verifies the release checksum, and installs
to `~/.local/bin`. Add this directory to `PATH` if it is not already there.
Run the same command to update.

### Windows

Run in PowerShell:

```powershell
Invoke-RestMethod https://raw.githubusercontent.com/esper-io/esper-cli/main/install.ps1 | Invoke-Expression
```

The script detects amd64 or arm64, verifies the release checksum, and installs
to `%LOCALAPPDATA%\Programs\espercli`. It adds this directory to your user `PATH`.
Run the same command to update.

### Installer options

Set `ESPERCLI_VERSION` to a stable version such as `2.0.1` to install that release.
Set `ESPERCLI_INSTALL_DIR` to change the install directory. Set these variables
in the shell that runs the script, not just the `curl` command.

The shell script also supports macOS and Windows through Git Bash. On macOS,
it uses Homebrew only when neither override is set. Explicit version or directory
settings always use a direct download. You can download and review either script
before you run it.

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

## Agent Guidance

[`SKILL.md`](SKILL.md) gives AI agents instructions for the CLI. It explains how
to select commands, read JSON, and get human approval. Automation generates this
file from the same command data as the CLI. Automated checks find differences
between the guide and the command data.

## Releases

Push a stable version tag, such as `v2.0.1`, from a commit on `main` to publish
a release. Do not use leading zeros in release tags. The release workflow runs
the Go tests and checks before it publishes the platform archives, checksums,
and Homebrew formula. Main-branch CI builds remain Actions artifacts.

## License

Apache-2.0. See [LICENSE](LICENSE).
