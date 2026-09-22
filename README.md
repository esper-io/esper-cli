# espercli

A command-line tool for the [Esper](https://esper.io) API — manage devices, applications, groups, pipelines, and more from your terminal.

Built with [Typer](https://typer.tiangolo.com) and [Rich](https://github.com/Textualize/rich) for a modern CLI experience: colour-coded output, tab-completion, inline help on errors, and auto-pagination.

Also ships a **Claude Code skill** (`/esper`) so you can manage your fleet in plain English directly from your AI coding session — no syntax required.

---

## Requirements

- Python 3.8+
- An Esper account with an API key ([generate one here](https://docs.esper.io/home/module/genapikey.html))

---

## Installation

**Recommended — pipx** (isolated, no venv management needed):

```sh
pipx install git+https://github.com/esper-io/esper-cli.git
```

**From source:**

```sh
git clone https://github.com/esper-io/esper-cli.git
cd esper-cli
pipx install -e .
```

**Plain pip** (inside a virtualenv):

```sh
pip install git+https://github.com/esper-io/esper-cli.git
```

---

## Quick start

```sh
# 1. Configure credentials (run once)
espercli configure

# 2. Check your active context at any time
espercli context

# 3. List your devices
espercli device list

# 4. Set a device as active for subsequent commands
espercli device show <name> --active

# 5. Explore any command with --help
espercli device --help
espercli device list --help
```

---

## Tab completion

Install once and restart your terminal:

```sh
espercli completion zsh        # zsh
espercli completion bash       # bash
espercli completion fish       # fish
espercli completion powershell # PowerShell
```

After installing, pressing `<Tab>` completes:

- **Sub-commands** — `espercli dev<TAB>` → `device`
- **Enum values** — `espercli device list --state <TAB>` → `active`, `inactive`, `disabled`, …
- **Device names** — `espercli device show <TAB>` → live names from your enterprise
- **Group names** — `espercli group-command reboot --group <TAB>`
- **Application IDs** — `espercli version list --app <TAB>` (shows app name as hint)
- **Command names, schedule types, days, and more**

---

## Commands

### `configure`
Set credentials (environment name and API key). Run this first.

```sh
espercli configure
```

---

### `device`
```sh
espercli device list                          # list all devices (paginated)
espercli device list --state active           # filter by state
espercli device list --group "Warehouse"      # filter by group
espercli device list --all                    # fetch every device (auto-paginate)

espercli device show <name>                   # show device details
espercli device show <name> --active          # show + set as active device

espercli device set-active --name <name>      # set active device
espercli device unset-active                  # clear active device

espercli device report <name>                 # HTML dashboard (opens in browser)
espercli device report <name> --output ./report.html   # write to specific path
espercli device report <name> --no-open       # write file without opening browser
```

---

### `device-command`
Fire commands at a single device (uses the active device if `--device` is omitted).

```sh
espercli device-command ping
espercli device-command lock    --device <name>
espercli device-command reboot  --device <name>
espercli device-command wipe    --device <name>          # requires confirmation
espercli device-command install --device <name> --version <version-id>
espercli device-command uninstall --device <name> --version <version-id>
espercli device-command clear-app-data --device <name> --package-name com.example.app
espercli device-command show <command-id>
```

---

### `group`
```sh
espercli group list
espercli group list --all                     # auto-paginate
espercli group show <name>
espercli group show <name> --active           # set as active group
espercli group create <name>
espercli group update <name> --name <new-name>
espercli group delete <name>                  # requires confirmation
espercli group devices                        # list devices in active group
espercli group add --group <name> --device <device-name>
espercli group remove --group <name> --device <device-name>
espercli group move --group <name> --parent <parent-name>
```

---

### `group-command`
Fire commands at every device in a group.

```sh
espercli group-command ping    --group <name>
espercli group-command lock    --group <name>
espercli group-command reboot  --group <name>
espercli group-command install --group <name> --version <version-id>
espercli group-command show <command-id>
```

---

### `app`
```sh
espercli app list
espercli app list --name "MyApp"
espercli app list --all                       # auto-paginate
espercli app show <app-id>
espercli app show <app-id> --active           # set as active application
espercli app upload <path/to/app.apk>
espercli app download <version-id> --dest ./app.apk
espercli app delete <app-id>                  # requires confirmation
espercli app set-active --id <app-id>
espercli app unset-active
```

---

### `version`
Manage versions of the active (or specified) application.

```sh
espercli version list
espercli version list --app <app-id>
espercli version list --legacy-format false   # show version_name instead of build_number
espercli version show <version-id>
espercli version delete <version-id>          # requires confirmation
espercli version devices <version-id>         # list devices with this version installed
```

---

### `installs`
```sh
espercli installs list                        # installs for active device
espercli installs list --device <name>
espercli installs show <install-id>
```

---

### `status`
```sh
espercli status show                          # status for active device
espercli status show --device <name>
espercli status list
```

---

### `commandsV2`
Multi-device / group command requests (V2 API).

```sh
# List recent requests
espercli commandsV2 list
espercli commandsV2 list --command reboot --command-type device

# Fire a command
espercli commandsV2 command \
  --command-type device \
  --devices "device-name-1 device-name-2" \
  --command reboot

# Check status / history
espercli commandsV2 status --request <request-id>
espercli commandsV2 history --device <name>
```

---

### `pipeline`
```sh
espercli pipeline list
espercli pipeline show <pipeline-id>
espercli pipeline create --name "Nightly Deploy" --no-of-stages 3
espercli pipeline edit <pipeline-id> --name "New Name"
espercli pipeline delete <pipeline-id>

# Stages
espercli pipeline stage list --pipeline <id>
espercli pipeline stage create --pipeline <id> --name "Stage 1"

# Operations
espercli pipeline stage operation create --stage <id> --action APP_INSTALL

# Execution
espercli pipeline execute start   --pipeline <id>
espercli pipeline execute stop    --pipeline <id>
espercli pipeline execute show    --pipeline <id>
```

---

### `content`
```sh
espercli content list
espercli content show <content-id>
espercli content upload <path/to/file>
espercli content modify <content-id> --tags "tag1 tag2" --description "..."
espercli content delete <content-id>       # requires confirmation
```

---

### `token`
```sh
espercli token show
espercli token renew
```

---

### `enterprise`
```sh
espercli enterprise show
espercli enterprise set-active
```

---

### `telemetry`
```sh
espercli telemetry get-data \
  --device <name> \
  --metric battery-level \
  --last 24 \
  --period hour \
  --statistic avg
```

---

### `secureadb`
```sh
espercli secureadb connect --device <name>
```

---

### Utility commands

```sh
espercli context          # show active environment, enterprise, device, app, group
espercli about            # show CLI version
espercli completion zsh   # install tab-completion
```

---

## Global flags

| Flag | Short | Description |
|---|---|---|
| `--verbose` | `-v` | Enable debug logging |
| `--no-color` | | Disable colour output (useful for piping / CI) |

---

## Output formats

Every listing and detail command supports `--json` / `-j` for machine-readable output:

```sh
espercli device list --json | jq '.[].id'
espercli app show <id> --json
```

---

## Error messages

Mistyped a command? The CLI shows the relevant help page inline — no need to re-run with `--help`:

```
$ espercli device lst

╭─ Error ────────────────────────────────────╮
│ ✗  No such command 'lst'. Did you mean 'list'?  │
╰────────────────────────────────────────────╯

 Usage: espercli device [OPTIONS] COMMAND [ARGS]...

╭─ Commands ─────────────────────────╮
│ list         List devices          │
│ show         Show device details   │
│ set-active   Set active device     │
│ unset-active Clear active device   │
╰────────────────────────────────────╯
```

---

## Claude Code skill (`/esper`)

The repo includes a [Claude Code](https://claude.ai/code) slash command that gives you a natural-language interface to your Esper fleet. Instead of remembering CLI syntax, just describe what you want.

### Setup

**Option A — project skill (automatic when you clone):**

The skill is already at `.claude/commands/esper.md` in this repo. Open the project in Claude Code and `/esper` is available immediately — no install step.

**Option B — user skill (available in every project):**

Copy the skill to your global commands directory so it works outside this repo too:

```sh
mkdir -p ~/.claude/commands
cp .claude/commands/esper.md ~/.claude/commands/esper.md
```

### Usage

Type `/esper` in Claude Code followed by what you want in plain English:

```
/esper show me all inactive devices
/esper reboot the warehouse group
/esper what apps are installed on ezra pixel
/esper upload build/app-release.apk and set it as the active app
/esper show battery telemetry for ezra pixel over the last 24 hours
/esper which devices have version abc-123 installed
```

Or just `/esper` with no argument — it checks your active context and waits for direction.

### What it does

- Runs `espercli context` first so it knows your active environment, device, and group
- Translates natural language into the right `espercli` commands and runs them
- Presents results as a clean summary, not raw CLI output
- Asks for confirmation before any destructive action (wipe, delete) regardless of how you phrase the request
- Asks a clarifying question if the target is ambiguous rather than guessing

---

## License

Apache 2.0 — see [LICENSE](LICENSE).
