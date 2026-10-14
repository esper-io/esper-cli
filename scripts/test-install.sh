#!/bin/sh
# Offline installer checks. Downloads and Homebrew are replaced with local fixtures.
set -eu
repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' 0
mkdir -p "$tmp/bin" "$tmp/fixture"
printf 'fixture binary\n' > "$tmp/fixture/espercli"
tar -czf "$tmp/archive.tar.gz" -C "$tmp/fixture" espercli
if command -v sha256sum >/dev/null 2>&1; then
    hash=$(sha256sum "$tmp/archive.tar.gz" | awk '{print $1}')
else
    hash=$(shasum -a 256 "$tmp/archive.tar.gz" | awk '{print $1}')
fi
cat > "$tmp/bin/uname" <<'SH'
#!/bin/sh
if [ "$1" = -s ]; then echo "$TEST_OS"; else echo "$TEST_ARCH"; fi
SH
cat > "$tmp/bin/brew" <<'SH'
#!/bin/sh
printf '%s\n' "$*" > "$TEST_ROOT/brew-called"
SH
cat > "$tmp/bin/sysctl" <<'SH'
#!/bin/sh
echo 0
SH
cat > "$tmp/bin/curl" <<'SH'
#!/bin/sh
out=
while [ "$#" -gt 0 ]; do
    case "$1" in -o) out=$2; shift ;; esac
    url=$1
    shift
done
case "$url" in
    */latest) echo latest >> "$TEST_ROOT/lookups"; printf 'https://github.com/esper-io/esper-cli/releases/tag/v2.0.1' ;;
    */checksums.txt) printf '%s  espercli_%s_%s_%s.tar.gz\n' "$TEST_HASH" "$TEST_ASSET_VERSION" "$TEST_ASSET_OS" "$TEST_ASSET_ARCH" > "$out" ;;
    */espercli_2.*) cp "$TEST_ROOT/archive.tar.gz" "$out" ;;
    *) exit 1 ;;
esac
SH
chmod +x "$tmp/bin/"*
export TEST_ROOT="$tmp" TEST_HASH="$hash" PATH="$tmp/bin:$PATH"
for platform in linux-x86 linux-arm64 darwin-version darwin-directory; do
    export TEST_OS=Linux TEST_ARCH=x86_64 TEST_ASSET_OS=linux TEST_ASSET_ARCH=x86 TEST_ASSET_VERSION=2.0.1
    unset ESPERCLI_VERSION ESPERCLI_INSTALL_DIR
    export HOME="$tmp/$platform"
    case "$platform" in
        linux-arm64) export TEST_ARCH=aarch64 TEST_ASSET_ARCH=arm64 ;;
        darwin-*) export TEST_OS=Darwin TEST_ASSET_OS=darwin ;;
    esac
    case "$platform" in
        darwin-version) export ESPERCLI_VERSION=v2.0.2 TEST_ASSET_VERSION=2.0.2 ;;
        darwin-directory) export ESPERCLI_INSTALL_DIR="$HOME/custom bin" ;;
    esac
    sh "$repo/install.sh"
    cmp "$tmp/fixture/espercli" "${ESPERCLI_INSTALL_DIR:-$HOME/.local/bin}/espercli"
    test ! -e "$tmp/brew-called"
done
unset ESPERCLI_VERSION ESPERCLI_INSTALL_DIR
sh "$repo/install.sh"
test "$(cat "$tmp/brew-called")" = 'install esper-io/tap/espercli'
export TEST_OS=Linux TEST_ASSET_OS=linux TEST_ASSET_ARCH=x86 TEST_HASH=bad ESPERCLI_INSTALL_DIR="$tmp/rejected"
if sh "$repo/install.sh"; then echo 'Expected checksum rejection' >&2; exit 1; fi
test ! -e "$tmp/rejected/espercli"
export ESPERCLI_VERSION=../../invalid
if sh "$repo/install.sh"; then echo 'Expected invalid version rejection' >&2; exit 1; fi
printf 'Installer checks passed.\n'
