#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' 0
cd "$tmp"
git init -q
git config user.name 'Version test'
git config user.email 'version-test@example.invalid'
mkdir -p internal/version
write_version() { printf 'package version\nvar (\n\tVersion = "%s"\n)\n' "$1" > internal/version/version.go; }
version() { sh "$repo/scripts/release-version.sh" "$@"; }
write_version 2.0.0
git add .
git commit -qm 'Version fixture'
previous=$(git rev-parse HEAD)
test "$(version source)" = 2.0.0
test -z "$(version bump "$previous")"
test "$(version ci 69 68)" = "$(printf 'CI_VERSION=2.0.01\nCI_SNAPSHOT_VERSION=2.0.1')"
write_version 2.0.1
test -z "$(version bump "$previous")"
write_version 2.1.0
test "$(version bump "$previous")" = 2.1.0
test "$(version ci 70 68)" = "$(printf 'CI_VERSION=2.1.02\nCI_SNAPSHOT_VERSION=2.1.2')"
test "$(version ci 168 68)" = "$(printf 'CI_VERSION=2.1.100\nCI_SNAPSHOT_VERSION=2.1.100')"
write_version 3.0.0
test "$(version bump "$previous")" = 3.0.0
write_version 1.9.0
if version bump "$previous"; then echo 'Expected downgrade rejection' >&2; exit 1; fi
write_version 2.01.0
if version source; then echo 'Expected invalid version rejection' >&2; exit 1; fi
write_version 2.0.0
if version ci 68 68; then echo 'Expected invalid offset rejection' >&2; exit 1; fi
write_version 2.2.0
git add .
git commit -qm 'New release line'
write_version 2.1.0
if version bump HEAD; then echo 'Expected minor downgrade rejection' >&2; exit 1; fi
printf 'Release version checks passed.\n'
