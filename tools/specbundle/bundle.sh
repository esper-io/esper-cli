#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  printf '%s\n' "usage: bundle.sh <esper-api-docs-checkout>" >&2
  exit 2
fi

repo=$1
root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/esper-openapi.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

npx --yes @redocly/cli@1.34.5 bundle "$repo/openapi.yaml" --output "$temporary_dir/source.json" --ext json
node "$root/tools/specbundle/main.mjs" "$temporary_dir/source.json" "$temporary_dir/output"
rm -f "$root/spec/openapi/"*.yaml "$root/spec/openapi/manifest.json"
cp "$temporary_dir/output/"*.yaml "$temporary_dir/output/manifest.json" "$root/spec/openapi/"
