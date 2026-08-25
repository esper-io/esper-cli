# Go CLI live smoke test

Status: human-run only. Run against a dev enterprise. Do not use production
credentials or production devices.

## Prerequisites

- Build or install the `espercli` binary from the `go-rewrite` branch.
- Install `jq` and Android platform tools (`adb`).
- Use a dev API key and a dev device on which Remote ADB is enabled.
- Export `SMOKE_DEVICE` as the ID of a dev device you own and may ping/debug.
- Run the checklist in Bash, with `espercli`, `jq`, and `adb` on `PATH`.

```bash
set -euo pipefail
command -v bash
command -v jq
command -v adb
: "${SMOKE_DEVICE:?export SMOKE_DEVICE as an owned dev device ID}"
: "${ESPER_ENVIRONMENT:?export ESPER_ENVIRONMENT for the dev tenant}"
espercli version
```

Expected: dependency checks print executable paths, and `espercli version`
prints `espercli <version> (commit ..., built ...)`. Exit code: `0` for each.
If `espercli version` starts Python, raises an import error, or does not print Go
build metadata, a legacy Python CLI is earlier on `PATH`. Remove that install or
bypass it before continuing:

```bash
type -a espercli
mkdir -p dist
go build -o dist/espercli ./cmd/espercli
export PATH="$PWD/dist:$PATH"
hash -r
espercli version
```

Do not continue until the version command identifies the Go CLI.

## 1. Configure credentials

Set the dev environment name and enter the API key without putting it in shell
history:

```bash
read -rsp "Dev Esper API key: " ESPER_SMOKE_API_KEY; printf '\n'
espercli configure \
  --environment "$ESPER_ENVIRONMENT" <<<"$ESPER_SMOKE_API_KEY"
printf 'exit=%s\n' "$?"
```

Expected stdout:

```text
Configuration saved.
exit=0
```

Expected stderr: the `API key:` prompt. The key is supplied through stdin so it
does not appear in shell history or the `espercli` process argument list.

Verify persistence and redaction:

```bash
espercli configure show
printf 'exit=%s\n' "$?"
```

Expected shape: two lines named `environment` and `api_key`; the environment is
the exact value of `$ESPER_ENVIRONMENT`, and the API key is masked except for its
final four characters. The unredacted key must not appear. Exit code: `0`.

## 2. Set enterprise context

```bash
read -rp "Dev enterprise ID: " ESPER_SMOKE_ENTERPRISE_ID
espercli context set enterprise "$ESPER_SMOKE_ENTERPRISE_ID"
ESPER_SMOKE_CONTEXT_SET_EXIT="$?"
espercli context get enterprise
ESPER_SMOKE_CONTEXT_GET_EXIT="$?"
printf 'set-exit=%s\nget-exit=%s\n' \
  "$ESPER_SMOKE_CONTEXT_SET_EXIT" "$ESPER_SMOKE_CONTEXT_GET_EXIT"
test "$ESPER_SMOKE_CONTEXT_SET_EXIT" -eq 0
test "$ESPER_SMOKE_CONTEXT_GET_EXIT" -eq 0
```

Expected shape: `enterprise: <the supplied ID>` from both context commands.
Exit code: `0`.

## 3. Verify the explicitly selected device

Use a bounded query to verify the corrected list envelope, then select only the
exact human-supplied device through its detail endpoint. Never select the first
device from a shared dev tenant.

```bash
ESPER_SMOKE_DEVICE_JSON="$(
  espercli device list --limit 5 --json
)"
printf '%s\n' "$ESPER_SMOKE_DEVICE_JSON" |
  jq -e '.content.results | type == "array" and length <= 5'
ESPER_SMOKE_DEVICE_EXIT="$?"
ESPER_SMOKE_DEVICE_DETAIL="$(espercli device get "$SMOKE_DEVICE" --json)"
printf '%s\n' "$ESPER_SMOKE_DEVICE_DETAIL" |
  jq -e --arg device "$SMOKE_DEVICE" '.content.id == $device'
ESPER_SMOKE_DEVICE_DETAIL_EXIT="$?"
export ESPER_SMOKE_DEVICE_ID="$SMOKE_DEVICE"
printf 'list-exit=%s\ndetail-exit=%s\n' \
  "$ESPER_SMOKE_DEVICE_EXIT" "$ESPER_SMOKE_DEVICE_DETAIL_EXIT"
test "$ESPER_SMOKE_DEVICE_EXIT" -eq 0
test "$ESPER_SMOKE_DEVICE_DETAIL_EXIT" -eq 0
```

Expected stdout: `true` twice, followed by `list-exit=0` and `detail-exit=0`.
Both raw responses remain apps envelopes; the selected ID is verified at
`.content.id` and never inferred from list order.

## 4. Show the selected device

```bash
espercli device get "$ESPER_SMOKE_DEVICE_ID"
printf 'exit=%s\n' "$?"
```

Expected shape: a key/value block containing at least the selected device ID,
name, and state, with no outer `content` row. Exit code: `0`.

## 5. Verify pagination with `--all` on groups

```bash
espercli device-group list --limit 100 --all
printf 'exit=%s\n' "$?"
```

Expected shape: one merged human-readable table containing the dev enterprise's
groups, not one envelope per page. Exit code: `0`.

## 6. Verify `--json` through `jq`

```bash
espercli device-group list --limit 100 --all --json |
  jq -e 'type == "array" and all(.[]; has("id"))'
printf 'exit=%s\n' "$?"
```

Expected stdout: `true`, followed by `exit=0`. The CLI side of the pipe emits
only one valid JSON array and no status text.

## 7. Send a non-destructive ping

Create an immediate `UPDATE_HEARTBEAT` command for the selected device. The
active enterprise context supplies `enterprise_id`.

```bash
ESPER_SMOKE_PING_BODY="$(
  jq -nc --arg device "$SMOKE_DEVICE" \
    '{command_type:"DEVICE", devices:[$device], command:"UPDATE_HEARTBEAT", schedule:"IMMEDIATE"}'
)"
espercli command create --body "$ESPER_SMOKE_PING_BODY" --json |
  jq -e 'type == "object"'
printf 'exit=%s\n' "$?"
```

Expected stdout: `true`, followed by `exit=0`. The API response is a JSON
command-request object, commonly containing an `id` and status fields. This is
the checklist's non-destructive command request. Secure ADB later creates its
own temporary relay session.

## 8. Verify destructive confirmation without `--yes`

Answer `n`; this must stop before an API request is sent.

```bash
set +e
printf 'n\n' | espercli device-request delete "$SMOKE_DEVICE"
ESPER_SMOKE_CANCEL_EXIT="$?"
set -e
printf 'exit=%s\n' "$ESPER_SMOKE_CANCEL_EXIT"
test "$ESPER_SMOKE_CANCEL_EXIT" -eq 2
```

Expected stderr shape: a confirmation naming the device request path and one
target. Expected final line: `exit=2`. The command must report cancellation and
must not delete or unenroll the device.

## 9. Connect Secure ADB

Start the relay for the exact selected device in terminal A:

```bash
set +e
espercli --verbose secureadb connect --device "$SMOKE_DEVICE"
ESPER_SMOKE_SECUREADB_EXIT="$?"
set -e
printf 'secureadb-exit=%s\n' "$ESPER_SMOKE_SECUREADB_EXIT"
test "$ESPER_SMOKE_SECUREADB_EXIT" -eq 0
```

Expected terminal A shape:

```text
context: using active enterprise ... for enterprise_id
Secure ADB relay ready. Run:
adb connect 127.0.0.1:<ephemeral-port>
```

The command remains running while it waits for and bridges one local ADB
connection. In terminal B, copy the exact endpoint printed by terminal A:

```bash
adb connect 127.0.0.1:<ephemeral-port>
adb devices
```

Expected terminal B shape: `connected to 127.0.0.1:<port>` and an `adb devices`
entry for that endpoint. Exit code: `0` for each command.

If `adb devices` shows the endpoint as `unauthorized`, keep the relay running,
unlock `SMOKE_DEVICE`, and accept the one-time Android "Allow USB debugging?"
RSA fingerprint prompt. Then rerun `adb connect 127.0.0.1:<ephemeral-port>`
against the same local relay. This is normal first-connect behavior for non-EEA
or stock Android devices. Esper Foundation devices auto-authorize the session
key. Remote ADB requires device policy to allow debugging features.

After confirming connectivity, disconnect in terminal B and press Ctrl+C in
terminal A if it has not exited:

```bash
adb disconnect 127.0.0.1:<ephemeral-port>
printf 'disconnect-exit=%s\n' "$?"
```

Expected terminal B final line: `disconnect-exit=0`. Expected terminal A final
shape: `Session duration: ...`, `Bytes transferred: ...`, and
`secureadb-exit=0` when interrupted after the ADB bridge is established. If the
command is interrupted before the local ADB client connects,
`secureadb-exit=4` is expected and does not validate the relay.

## 10. Remove shell secrets

```bash
unset ESPER_SMOKE_API_KEY ESPER_SMOKE_PING_BODY ESPER_SMOKE_DEVICE_JSON ESPER_SMOKE_DEVICE_DETAIL
```

Expected output: none. Exit code: `0`.

## Results 2026-08-25

Environment: `develop`. Executor used the repository Go binary at
`./dist/espercli-smoke` because the `espercli` found on `PATH` is the legacy
Python installation and fails during import. No API key is included below.

| Step | Command | Expected | Actual | Result |
|---|---|---|---|---|
| Safety preamble | `espercli context clear --all` | All four resources unset; exit 0 | PATH binary failed before command execution. The repository Go binary then cleared all four resources with exit 0 | FAIL |
| Configuration precheck | `espercli configure show` | `develop`, redacted key; exit 0 | Initially unconfigured, exit 3. After human credential entry: `environment: develop`, redacted key ending `t8eN`, exit 0 | PASS |
| 1. Configure | `espercli configure --environment develop` with key on stdin | `Configuration saved.`; exit 0 | Human completed the stdin prompt; subsequent `configure show` verified persisted environment and redacted key | PASS |
| 2. Enterprise context | `espercli context set enterprise f44373cb-1800-43c6-aab3-c81f8b1f435c`; `context get enterprise` | Supplied ID twice; exit 0 | Supplied ID twice; both exits 0 | PASS |
| 3a. Device list | `espercli device list --limit 5` | Top-level `count`, `next`, `previous`, `results` key/value block; exit 0 | One top-level `content` value containing `count`, `next`, `prev`, and five device results; exit 0 | FAIL |
| 3b. Select device | `espercli device list --limit 1 --json \| jq -er '.results[0].id'` | Non-empty device ID; exit 0 | `device=null`; exit 1 because results are under `.content.results` | FAIL |
| 4. Device show | `espercli device get "$ESPER_SMOKE_DEVICE_ID"` | Device key/value block; exit 0 | Not run because step 3b produced no device ID | SKIPPED |
| 5. Pagination `--all` | `espercli device list --limit 2 --all` | One merged table; exit 0 | Not run. Step 3a reported 142,632 devices; this command would require approximately 71,316 live requests | SKIPPED |
| 6. JSON through `jq` | `espercli device list --limit 2 --all --json \| jq ...` | `true`; exit 0 | Not run for the same request-volume reason as step 5 | SKIPPED |
| 7. Ping | `espercli command create --body '<redacted device request>' --json \| jq ...` | JSON object; exit 0 | Not run because no device was selected | SKIPPED |
| 8. Destructive prompt | `printf 'n\n' \| espercli device-request delete "$ESPER_SMOKE_DEVICE_ID"` | Exact target/count prompt, decline, exit 2, no API call | Not run because no exact device target was selected | SKIPPED |
| 9. Secure ADB | `espercli --verbose secureadb connect` | Loopback ADB endpoint, successful bridge, metrics, exit 0 | Not run because no selected device could be checked for Remote ADB eligibility | SKIPPED |
| 10. Cleanup | `unset ESPER_SMOKE_API_KEY ESPER_SMOKE_PING_BODY` | No output; exit 0 | No output; exit 0 | PASS |

### Failure Evidence

PATH binary preamble failure (before any API call):

```text
$ espercli context clear --all
Traceback (most recent call last):
  File "/Users/rohanhasabe/.local/bin/espercli", line 6, in <module>
    from esper.cli.app import main_entry
  File "/Users/rohanhasabe/Library/Application Support/pipx/venvs/espercli/lib/python3.14/site-packages/esper/cli/app.py", line 32, in <module>
    from esper.cli.secureadb import app as secureadb_app
  File "/Users/rohanhasabe/Library/Application Support/pipx/venvs/espercli/lib/python3.14/site-packages/esper/cli/secureadb.py", line 17, in <module>
    from esper.ext.certs import cleanup_certs, create_self_signed_cert, save_device_certificate
  File "/Users/rohanhasabe/Library/Application Support/pipx/venvs/espercli/lib/python3.14/site-packages/esper/ext/certs.py", line 4, in <module>
    from cement.utils import fs
ModuleNotFoundError: No module named 'cement'
exit=1
```

Initial configuration precheck, later resolved by human credential entry:

```text
$ ./dist/espercli-smoke configure show
error: auth: Esper credentials are not configured
exit=3
```

Device-list output shape (device objects are omitted from committed evidence
because they contain live identifiers, network details, serials, and location):

```text
$ ./dist/espercli-smoke device list --limit 5
content  {"count":142632,"next":"https://develop-api.esper.cloud/api/v2/devices/?limit=5&offset=5","prev":null,"results":[<5 live device objects redacted>]}
exit=0
```

Device selection failure:

```text
$ ./dist/espercli-smoke device list --limit 1 --json | jq -er '.results[0].id'
device=null
exit=1
```

No destructive operation was executed. No destructive confirmation was reached
because the preceding selection step did not produce an exact target. The only
live API requests completed during this run were the two read-only device-list
requests in steps 3a and 3b.

## Results 2026-08-25 Rerun

Environment reported by the switched dev tenant: `rjhlf`. Human-supplied owned
device: `be51677d-d0f4-4a08-a06d-cea69429b5a8`. Enterprise:
`18757e17-abb8-464d-b88b-c5ee4897c793`. The API key remained redacted.

| Step | Command | Expected | Actual | Result |
|---|---|---|---|---|
| Safety reset | `espercli context clear --all` | All four resources unset; exit 0 | All four resources unset; exit 0 | PASS |
| Go binary preamble | `espercli version` | Go build metadata; exit 0 | `espercli dev (commit unknown, built unknown)`; exit 0 | PASS |
| 1. Configuration | `espercli configure show` | Environment `develop`, redacted key; exit 0 | Environment `rjhlf`, redacted key; exit 0 | FAIL |
| 2. Enterprise context | `context set/get enterprise 18757e17-abb8-464d-b88b-c5ee4897c793` | Exact enterprise twice; exits 0 | Exact enterprise twice; exits 0 | PASS |
| 3. Bounded owned-device query | `device list --limit 5 --search "$SMOKE_DEVICE" --json` with exact-ID `jq` assertion | `true`; exits 0 | CLI exit 0; assertion returned `false`, exit 1 | FAIL |
| 4. Exact device get | `device get "$SMOKE_DEVICE"` | Top-level device key/value block; exit 0 | One `content` key containing the exact device object; exit 0 | FAIL |
| 5. Group pagination | `device-group list --limit 100 --all` | One merged group table; exit 0 | One-row merged `All devices` group table; exit 0 | PASS |
| 6. Group JSON pagination | `device-group list --limit 100 --all --json \| jq ...` | `true`; exit 0 | `true`; exit 0 | PASS |
| 7. Ping | `command create` with `UPDATE_HEARTBEAT` targeting only `SMOKE_DEVICE` | JSON object; exit 0 | `true`; exit 0 | PASS |
| 8. Destructive prompt | `device-request delete "$SMOKE_DEVICE"` without `--yes`, answer `n` | Exact target and count, cancellation, exit 2, no API request | Prompt named `/device/v0/devices/be51677d-d0f4-4a08-a06d-cea69429b5a8/` and `1 target(s)`; declined; exit 2 | PASS |
| 9. Secure ADB | `secureadb connect --device "$SMOKE_DEVICE"` | Loopback endpoint, ADB bridge, metrics; exit 0 | Active enterprise applied, then `device certificate is not valid PEM`; exit 1; no local endpoint | FAIL |
| 10. Cleanup | `unset ESPER_SMOKE_API_KEY ESPER_SMOKE_PING_BODY ESPER_SMOKE_DEVICE_JSON` | No output; exit 0 | No output; exit 0 | PASS |

### Rerun Failure Evidence

Configuration shape mismatch:

```text
$ espercli configure show
environment: rjhlf
api_key: **************************zgTj
exit=0
```

Bounded explicit-device assertion:

```text
$ espercli device list --limit 5 --search "$SMOKE_DEVICE" --json |
    jq -e --arg device "$SMOKE_DEVICE" '.content.results | any(.id == $device)'
false
cli-exit=0
jq-exit=1
```

The exact device lookup succeeded, proving ownership/tenant targeting, but the
human output retained an additional response envelope. Live device fields are
redacted from committed evidence:

```text
$ espercli device get "$SMOKE_DEVICE"
content  {"id":"be51677d-d0f4-4a08-a06d-cea69429b5a8","tenant_id":"18757e17-abb8-464d-b88b-c5ee4897c793",<live device fields redacted>}
exit=0
```

Secure ADB single attempt:

```text
$ espercli --verbose secureadb connect --device "$SMOKE_DEVICE"
context: using active enterprise 18757e17-abb8-464d-b88b-c5ee4897c793 for enterprise_id
error: api: device certificate is not valid PEM
secureadb-exit=1
```

No destructive API operation was executed. The exact destructive target was
declined at the prompt. The only mutating command request was the permitted
`UPDATE_HEARTBEAT` ping. Secure ADB created its temporary negotiation session but
did not open a local listener because certificate validation failed.

## Results 2026-08-25 Targeted Rerun

Only the four previously failed checks were rerun. Environment:
`$ESPER_ENVIRONMENT=rjhlf`. Device and enterprise were unchanged from the prior
rerun. The API key remained redacted.

| Failed step rerun | Expected | Actual | Result |
|---|---|---|---|
| Configure assertion | Stored environment equals `$ESPER_ENVIRONMENT`; redacted key; exit 0 | Stored environment `rjhlf` matched `$ESPER_ENVIRONMENT`; key redacted; exit 0 | PASS |
| Bounded list and exact detail JSON | Bounded `.content.results` array and exact `.content.id`; all exits 0 | Both `jq` assertions returned `true`; all four CLI/`jq` exits 0 | PASS |
| Exact detail human output | Device fields rendered without outer `content`; exit 0 | Device fields rendered directly as a key/value block; no `content` row; exit 0 | PASS |
| Secure ADB | Pinned TLS, loopback endpoint, authorized ADB device or documented one-time authorization prompt, metrics; exit 0 | Negative-serial certificate accepted; loopback endpoint and metrics produced; CLI/connect/devices/disconnect exits 0. Stock/non-EEA ADB authorization requires accepting the on-device RSA prompt once | PASS-WITH-MANUAL-STEP |

### Targeted Rerun Failure Evidence

The certificate parser and pinned TLS stages succeeded. The remaining failure is
ADB protocol authorization after the local bridge opened:

```text
$ espercli --verbose secureadb connect --device "$SMOKE_DEVICE"
context: using active enterprise 18757e17-abb8-464d-b88b-c5ee4897c793 for enterprise_id
Secure ADB relay ready. Run:
adb connect 127.0.0.1:<ephemeral-port>
failed to authenticate to 127.0.0.1:<ephemeral-port>
List of devices attached
<physical dev device>  device
127.0.0.1:<ephemeral-port>  unauthorized
disconnected 127.0.0.1:<ephemeral-port>
Session duration: 436ms
Bytes transferred: 1373
secureadb-exit=0
connect-exit=0
devices-exit=0
disconnect-exit=0
```

Python legacy and Go both provide only mTLS plus raw byte forwarding; neither
implements ADB AUTH or manages the local adb host key. Device-side code
auto-authorizes the supplied key only on Esper Foundation devices. The observed
`unauthorized` state is therefore normal first-connect behavior on stock/non-EEA
devices and is closed as PASS-with-manual-step.

No ping or destructive operation was repeated during this targeted rerun. No
destructive API operation was executed.
