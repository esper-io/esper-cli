# CLI API consolidation

## Goal and scope

Use newer APIs where they are demonstrably interchangeable, retain APIs that
serve different purposes, and prefix confirmed iOS-only commands with `ios-`.
Compare ambiguous endpoints before deciding whether an older API can be removed.

User choices: no ticket; task slug `cli-api-consolidation`; `ios-<resource>` names.
The current agent may perform this work directly. The earlier Luna-only testing
workflow has ended. Its CSV and evidence remain historical verification records.

## Plan

1. Inventory endpoint pairs and platform contracts from canonical specifications.
2. Run bounded, read-only comparisons against the configured develop tenant using
   the CLI, preserving request arguments, timing, errors, response shapes, and
   comparisons without persisting sensitive response values.
3. Classify each pair as compatible, meaningfully different, or unresolved.
   A successful GET does not prove compatibility of PATCH, POST, or DELETE.
4. Implement confirmed command changes through source generation, not manual
   edits to generated files. Keep unresolved APIs available pending a decision.
5. Update routing tests, generated command documentation, and migration guidance.
6. Run focused tests, contract checking, full Go tests, vet, and build checks.

## Compatibility criteria

Compare resource identity and scope, accepted parameters, filters, pagination,
response envelopes and fields, and operation effects. Similar names and higher
version numbers alone are insufficient. Empty lists and error responses cannot
establish response equivalence. Differences are recorded in `decisions.md`.

For write comparisons, inspect contracts first and request the CLI's normal human
approval for each actual mutation. Existing resources may be read. Mutations
remain limited to the previously authorized devices/blueprint or newly created
fixtures; no new writes are needed for the initial comparison batch.

## Planned files and diff budget

Approximately 800–1,200 hand-written changed lines, plus regenerated output.
Stop and revise with the user if the work exceeds approximately 2,400 lines.

- `.spec/cli-api-consolidation/{spec,decisions,rationale,learnings}.md`
- `.spec/cli-api-consolidation/compare_endpoints.py`
- `.spec/cli-api-consolidation/comparisons.jsonl` (sanitized live evidence)
- `tools/codegen/main.go` and `tools/codegen/main_test.go`
- `internal/commandpolicy/policy.go` (explicit, shared naming and replacement rules)
- `tools/contractcheck/main.go` (validate intentional replacement aliases)
- `internal/cmd/generated/runner.go` or group-description source, if required
- Relevant existing command golden tests under `internal/cmd/`
- `internal/cmd/api_consolidation_test.go` (public commands, retained routes, help)
- `README.md`
- Regenerated `internal/cmd/generated/zz_generated_commands.go`
- Regenerated `.claude/commands/esper.md`

Any additional policy/contract-check file changes will be listed before editing.
Preserve user-owned `.gitignore`, `.DS_Store`, `coverage.out`, and the untracked
`esper/espercli-commands.md`. Do not change old verification results.

## Acceptance criteria

- Confirmed iOS-only commands have the agreed prefix and retain correct routes.
- Cross-platform APIs do not acquire an iOS-only label.
- Different resources and unresolved counterparts remain reachable.
- Every removal or redirect has explicit compatibility evidence and tests.
- Generated metadata and documentation reproduce deterministically.
- Live comparisons distinguish API failures, local usage errors, and missing
  fixtures; no claims of compatibility based solely on help or status codes.
- No secrets, downloaded files, raw device details, or personal user data enter
  the comparison evidence.

## Implementation decisions after comparison

- Retire only `api legacy installdevice list`, mapping its metadata to the
  existing `getInstallDevicesV1` canonical alias. The runtime already excludes
  alias endpoints, so `installdevice list` uses only the newer backend. The old
  command spelling is removed; README documents its replacement.
- Do not retire the other compared endpoints. In particular, event feeds have
  different documented pagination defaults and required response fields despite
  matching records in the sample.
- Prefix the explicitly iOS app catalog, iTunes app metadata, webclip, and
  provisioning-profile operations. Move the provisioning-profile variant of
  `version list` to `ios-provisioning-profile-version list`, retaining its scope
  flag. Keep blueprint and tenant-app version routes platform-neutral.
- Apply naming after generation-based routing, preserving unrelated historical
  command paths such as `api legacy app list`. Keep original resource nouns and
  endpoint metadata for classification and fixture coverage.
- Apple-wide APNs, DEP, VPP, and Apple App Store search APIs retain their names
  pending evidence that their supported platform scope is exclusively iOS.
