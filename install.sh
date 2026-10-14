#!/bin/sh
# Install Esper CLI. Overrides always use a direct, checksum-verified download.
# ESPERCLI_VERSION: release version, with or without v (default: latest stable).
# ESPERCLI_INSTALL_DIR: binary directory (default: ~/.local/bin).
set -eu

fail() { printf 'espercli-install: %s\n' "$*" >&2; exit 1; }

main() {
    case "$(uname -s)" in
        Linux) os=linux ;;
        Darwin) os=darwin ;;
        MINGW*|MSYS*|CYGWIN*) os=windows ;;
        *) fail "Unsupported operating system: $(uname -s)" ;;
    esac
    if [ "$os" = darwin ] && [ -z "${ESPERCLI_VERSION:-}" ] &&
        [ -z "${ESPERCLI_INSTALL_DIR:-}" ] && command -v brew >/dev/null 2>&1; then
        brew install esper-io/tap/espercli </dev/null
        return
    fi
    case "$(uname -m)" in
        x86_64|amd64) arch=x86 ;;
        aarch64|arm64) arch=arm64 ;;
        *) fail "Unsupported architecture: $(uname -m)" ;;
    esac
    if [ "$os" = darwin ] && [ "$arch" = x86 ] &&
        [ "$(sysctl -n sysctl.proc_translated 2>/dev/null || true)" = 1 ]; then
        arch=arm64
    fi
    for tool in curl awk mktemp; do
        command -v "$tool" >/dev/null 2>&1 || fail "Required command not found: $tool"
    done
    version=${ESPERCLI_VERSION:-}
    if [ -z "$version" ]; then
        url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' https://github.com/esper-io/esper-cli/releases/latest)
        case "$url" in
            https://github.com/esper-io/esper-cli/releases/tag/v*) version=${url##*/tag/} ;;
            *) fail "Cannot find the latest stable release." ;;
        esac
    fi
    version=${version#v}
    printf '%s\n' "$version" | awk '/^2\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/ { ok=1 } END { exit !ok }' || fail "Expected a stable version such as 2.0.1."
    ext=tar.gz
    binary=espercli
    if [ "$os" = windows ]; then ext=zip; binary=espercli.exe; fi
    asset="espercli_${version}_${os}_${arch}.${ext}"
    base="https://github.com/esper-io/esper-cli/releases/download/v${version}"
    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' 0
    trap 'exit 1' HUP INT TERM
    curl -fsSL -o "$tmp/$asset" "$base/$asset"
    curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt"
    expected=$(awk -v name="$asset" '$2 == name { print $1 }' "$tmp/checksums.txt")
    [ -n "$expected" ] || fail "No checksum found for $asset."
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$tmp/$asset" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
    else
        fail "Install sha256sum or shasum to verify the download."
    fi
    [ "$actual" = "$expected" ] || fail "Checksum mismatch for $asset. Nothing was installed."
    if [ "$os" = windows ]; then
        unzip -q "$tmp/$asset" "$binary" -d "$tmp"
    else
        tar -xzf "$tmp/$asset" -C "$tmp" "$binary"
    fi
    dir=${ESPERCLI_INSTALL_DIR:-$HOME/.local/bin}
    mkdir -p "$dir"
    # Rename on the destination filesystem, without overwriting a running binary.
    staged=$(mktemp "$dir/.espercli.XXXXXX")
    if ! cp "$tmp/$binary" "$staged" || ! chmod 755 "$staged" || ! mv -f "$staged" "$dir/$binary"; then
        rm -f "$staged"
        fail "Cannot install to $dir."
    fi
    printf 'Installed Esper CLI %s to %s/%s\n' "$version" "$dir" "$binary"
    case ":$PATH:" in
        *":$dir:"*) ;;
        *) printf 'Add this directory to PATH: export PATH="%s:$PATH"\n' "$dir" ;;
    esac
}

# Do not start an install until the entire script has downloaded.
main
