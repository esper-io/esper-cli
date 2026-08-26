# Command Harness

The harness is designed for repeatable runs by CI or Luna. It never uses live
credentials in command arguments. The CLI reads credentials from the normal
state file or `ESPER_CREDS_FILE`.

## Modes

### Offline

Checks every generated operation and every hand-written command by resolving its
Cobra path and rendering help. Aliases are counted separately. No network or
credentials are used.

```bash
./scripts/test-commands.sh offline
```

Report: `dist/command-harness-offline.json`.

### Live read-only

Runs generated GET operations once with bounded inputs. It never adds `--all`
and never invokes POST, PUT, PATCH, DELETE, `--yes`, or Secure ADB. Missing
resource IDs are reported as `SKIPPED`. Collection routes without a `--limit`
parameter are also skipped because the harness will not issue an unbounded list
request.

To run those lists using each API's default page size, set
`HARNESS_ALLOW_DEFAULT_PAGE_LISTS=1`. This remains read-only and never adds
`--all`.

```bash
export HARNESS_ENTERPRISE=<dev-enterprise-id>
export HARNESS_DEVICE=<owned-dev-device-id>
./scripts/test-commands.sh live-readonly
```

Optional dependent IDs:

```bash
export HARNESS_GROUP=<safe-group-id>
export HARNESS_APP=<safe-application-id>
```

Report: `dist/command-harness-live-readonly.json`.

### Live mutations

Mutation runs are disabled by default. They require:

- `--allow-mutations` through the wrapper.
- A disposable enterprise ID in `HARNESS_ENTERPRISE` for tenant-scoped scenarios.
- An owned disposable device ID in `HARNESS_DEVICE` for device-scoped scenarios.
- `HARNESS_MUTATION_CONFIRM=I_UNDERSTAND_THIS_DEVICE_IS_DISPOSABLE` for device-scoped scenarios, or `I_UNDERSTAND_THIS_TENANT_IS_DISPOSABLE` for tenant-scoped scenarios.
- A scenario with at least one cleanup step.

The example removes a disposable iOS, Linux, or Windows device and clears context afterward.
Do not use it with Android devices.
The wrapper defaults to `testdata/command-harness/non-android-device-disposable.example.json`.

```bash
export HARNESS_ENTERPRISE=<disposable-enterprise-id>
export HARNESS_DEVICE=<disposable-device-id>
export HARNESS_MUTATION_SCOPE=device
export HARNESS_MUTATION_CONFIRM=I_UNDERSTAND_THIS_DEVICE_IS_DISPOSABLE
./scripts/test-commands.sh live-mutations
```

Mutation scenarios are JSON. Each step must declare `expected_exit`; every
non-GET generated operation must set `mutation: true`, and destructive generated
operations must include `--yes`. Cleanup runs in reverse order even when a setup
or mutation step fails. The example is intentionally not run by CI.
Device-scoped scenarios must target only the exact disposable device. The removal scenario
supports iOS, Linux, and Windows devices only; Android fixtures must not run it.

## Luna

Luna can run the same wrapper without prompts:

```bash
./scripts/test-commands.sh offline
./scripts/test-commands.sh live-readonly
```

For mutations, Luna must receive the disposable IDs and confirmation through
environment variables. Do not pass an API key, token, or request body secret on
the command line. The harness emits tab-separated progress and writes a JSON
report under `dist/`.
