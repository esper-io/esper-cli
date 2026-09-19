# Rationale

The current generator selects a root command by API generation after grouping
operations by noun and verb. That can make unrelated APIs look like replacements:
device events versus command statuses, device inventory versus operation targets,
or installed applications versus an iOS application catalog.

Platform naming must also be operation-specific. A shared `version list` command
contains blueprint, provisioning-profile, and tenant-app routes. Prefixing the
whole command would incorrectly label cross-platform variants.

Prefer a small, explicit command-routing policy with regression tests over broad
heuristics based on tags, generation names, or any occurrence of the word Apple.
Preserve endpoint metadata and generate outputs through the existing toolchain.

Live evidence will record shapes and aggregate equality/difference results, not
raw payloads. This permits comparing responses without saving credentials,
download links, user records, or device identifiers beyond the authorized scope.
