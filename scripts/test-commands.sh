#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-offline}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p dist
go build -o dist/espercli-harness-target ./cmd/espercli

case "$MODE" in
offline)
  go run ./tools/command-harness \
    --mode offline \
    --binary "$ROOT/dist/espercli-harness-target" \
    --report "$ROOT/dist/command-harness-offline.json"
  ;;
live-readonly)
  : "${HARNESS_ENTERPRISE:?set HARNESS_ENTERPRISE to the current dev enterprise ID}"
  : "${HARNESS_DEVICE:?set HARNESS_DEVICE to an owned dev device ID}"
  default_page_list_flag=()
  if [[ "${HARNESS_ALLOW_DEFAULT_PAGE_LISTS:-}" == "1" ]]; then
    default_page_list_flag=(--allow-default-page-lists)
  fi
  go run ./tools/command-harness \
    --mode live-readonly \
    --binary "$ROOT/dist/espercli-harness-target" \
    --enterprise "$HARNESS_ENTERPRISE" \
    --device "$HARNESS_DEVICE" \
    --group "${HARNESS_GROUP:-}" \
    --app "${HARNESS_APP:-}" \
    "${default_page_list_flag[@]}" \
    --report "$ROOT/dist/command-harness-live-readonly.json"
  ;;
live-mutations)
  : "${HARNESS_ENTERPRISE:?set HARNESS_ENTERPRISE to the disposable enterprise ID}"
  : "${HARNESS_DEVICE:?set HARNESS_DEVICE to the disposable device ID}"
  : "${HARNESS_MUTATION_CONFIRM:?set HARNESS_MUTATION_CONFIRM=I_UNDERSTAND_THIS_DEVICE_IS_DISPOSABLE}"
  scenario="${HARNESS_SCENARIO:-testdata/command-harness/non-android-device-disposable.example.json}"
  mutation_scope="${HARNESS_MUTATION_SCOPE:-device}"
  mutation_scope_flag=()
  if [[ "$mutation_scope" == "device" ]]; then
    mutation_scope_flag=(--disposable-device)
  fi
  go run ./tools/command-harness \
    --mode live-mutations \
    --binary "$ROOT/dist/espercli-harness-target" \
    --enterprise "$HARNESS_ENTERPRISE" \
    --device "$HARNESS_DEVICE" \
    --scenario "$scenario" \
    --allow-mutations \
    --confirmation "$HARNESS_MUTATION_CONFIRM" \
    "${mutation_scope_flag[@]}" \
    --report "$ROOT/dist/command-harness-live-mutations.json"
  ;;
*)
  printf 'usage: %s [offline|live-readonly|live-mutations]\n' "$0" >&2
  exit 2
  ;;
esac
