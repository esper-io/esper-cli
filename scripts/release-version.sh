#!/bin/sh
# Read the release line from the CLI source; CI's last component is a build counter.
set -eu
file=internal/version/version.go
read_version() {
    sed -nE 's/^[[:space:]]*Version = "([0-9]+\.[0-9]+\.[0-9]+)"$/\1/p' |
        awk '/^[1-9][0-9]*\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/ { print; ok=1 } END { exit !ok }'
}
version=$(read_version < "$file")
line=${version%.*}
case "${1:-source}" in
    source) printf '%s\n' "$version" ;;
    ci)
        build=$((${2:?workflow run number required} - ${3:?run offset required}))
        [ "$build" -gt 0 ] || { echo 'Run offset must be less than the workflow run number.' >&2; exit 1; }
        printf 'CI_VERSION=%s.%02d\nCI_SNAPSHOT_VERSION=%s.%d\n' "$line" "$build" "$line" "$build"
        ;;
    bump)
        previous=$(git show "${2:?previous commit required}:$file" | read_version)
        old_line=${previous%.*}
        [ "$line" != "$old_line" ] || exit 0
        major=${line%.*}; minor=${line#*.}
        old_major=${old_line%.*}; old_minor=${old_line#*.}
        if [ "$major" -lt "$old_major" ] || { [ "$major" -eq "$old_major" ] && [ "$minor" -lt "$old_minor" ]; }; then
            echo 'The release major/minor version must increase.' >&2
            exit 1
        fi
        printf '%s\n' "$version"
        ;;
    *) echo 'Use source, ci <run> <offset>, or bump <previous commit>.' >&2; exit 1 ;;
esac
