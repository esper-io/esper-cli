# Esper CLI Tool Change History

## 2.0.0 (Unreleased)

Esper CLI 2.0 is a new Go implementation for people and AI agents. It is not
an update from the Python CLI. Existing Python commands and scripts may need
changes. The Python CLI remains available on `develop`; the Python SDK is
unchanged.

### Added

- Standalone binaries for macOS, Linux, and Windows on amd64 and arm64.
- Commands generated using automation from reviewed public API definitions.
- Device, app, blueprint, convergence, pipeline, content, custom-action,
  reporting, enrollment, Apple, user, role, and permission commands.
- A consistent `espercli <singular-noun> <verb>` command format, built-in help,
  shell completion, scalar flags, and JSON bodies from inline text, files, or stdin.
- Local command discovery by name, API operation, path, request field, and
  documentation reference.
- A generated `SKILL.md` guide for AI agents.
- Tenant configuration and persistent active resource context. Explicit IDs
  override the active context.
- Human-readable tables for API responses, including nested fields and wrapped
  lists. Use `--json` to keep the complete response for scripts and agents.
- Pagination with `--all`, shared retries for supported request methods, and
  stable exit codes for API, usage, configuration, and network errors.
- Human approval for API writes. Each approval applies to the exact request
  and environment, expires after 15 minutes, and can be used only once.
- A separate confirmation for destructive operations. `--yes` skips that
  confirmation only; it does not replace human approval.
- Secure ADB relay setup and local ADB public-key submission for host
  pre-authorization on supported devices. Key selection supports
  `ESPER_ADB_PUB_KEY`, `ADB_VENDOR_KEYS`, and the default Android host key.
- Automated contract checks, command tests, and an offline command harness.
  Offline API fixtures are synthetic test data, not captured tenant responses
  or proof of live device behavior.
- A separate `go test ./...` CI check. Snapshot packaging depends on tests and
  verification passing. CI runs on pull requests and pushes to `main`.
- CI build versions starting at `2.0.01` and increasing with each workflow run.
  GoReleaser archive versions use the unpadded semantic version, such as `2.0.1`.
- Theme-aware Esper branding in the README.

### Changed

- Supported device commands use DSO through `operation create`. Legacy command
  submission remains only for the reviewed operation types DSO does not support.
- Removed direct `converge create`, `pipeline-command create/update`, and
  `command-status update` commands. Read-only history remains available.
- Removed superseded or unsupported API command routes. Pipeline execution
  still uses the backend's existing routing; this CLI change does not migrate it.
- Removed the Python implementation and its packaging from this branch.

### Fixed

- Pagination links no longer repeat the API base path.
- EMM web-token detail commands use the canonical trailing-slash route so
  redirects do not turn update or delete requests into reads.
- Switching tenants or enterprises clears stale active resource context.
  Rotating an API key for the same tenant and enterprise preserves context.
- Secure ADB accepts trusted device certificates with negative serial numbers,
  matching the Python CLI. Certificate signatures, trust chains, expiry, and
  server-auth usage remain verified.

## Python CLI history

## 0.0.1

Initial release.


## 0.0.12

Updated requests library to version 2.26.0

## 0.0.13

Adds support for:
- List devices with the app version installed
- Commands V2
- Content Management

Updated dependencies

## 0.0.14

- Add support for new device states

## 0.0.15

- Added App version format changes (includes version_name and version_code)
- Added `legacy_format {true/false}` cmd arg for app version format chnages

## 0.0.16

- Updated requirements.txt (esperclient==0.1.3)
