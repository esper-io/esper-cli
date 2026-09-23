#!/bin/sh
set -eu

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
  printf '%s\n' "usage: bundle.sh <esper-api-docs-checkout> <public-oas-url-or-path> [canonical-overlay-dir]" >&2
  exit 2
fi

repo=$1
root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
overlay_dir=${3:-"$root/spec/openapi"}
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/esper-openapi.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

npx --yes @redocly/cli@1.34.5 bundle "$repo/openapi.yaml" --output "$temporary_dir/source.json" --ext json
public_oas=$2
case "$public_oas" in
  http://*|https://*)
    curl --fail --location --silent --show-error "$public_oas" --output "$temporary_dir/public-oas.json"
    public_oas="$temporary_dir/public-oas.json"
    ;;
esac
node "$root/tools/specbundle/main.mjs" "$temporary_dir/source.json" "$temporary_dir/output" "$public_oas" "$overlay_dir"
rm -f "$root/spec/openapi/"*.yaml "$root/spec/openapi/manifest.json"
cp "$temporary_dir/output/"*.yaml "$temporary_dir/output/manifest.json" "$root/spec/openapi/"
