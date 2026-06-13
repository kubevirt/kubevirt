#!/usr/bin/env bash

set -e -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RPM_DEPS="${SCRIPT_DIR}/rpm-deps.sh"
REPO_URL="http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os"

dry_run=false
check_only=false

for arg in "$@"; do
    case "$arg" in
        --dry-run) dry_run=true ;;
        --check) check_only=true ;;
        -h|--help) echo "Usage: $0 [--dry-run|--check]"; exit 0 ;;
        *) echo "Unknown argument: $arg" >&2; exit 1 ;;
    esac
done

[[ -f "$RPM_DEPS" ]] || { echo "Cannot find $RPM_DEPS" >&2; exit 1; }

primary_href=$(curl -sL "${REPO_URL}/repodata/repomd.xml" |
    grep -oP 'href="\Krepodata/[^"]*primary\.xml\.gz') || true
[[ -n "$primary_href" ]] || { echo "repomd.xml: cannot find primary.xml.gz" >&2; exit 1; }

metadata=$(mktemp)
trap 'rm -f "$metadata"' EXIT
curl -sL "${REPO_URL}/${primary_href}" | gunzip >"$metadata"

query_latest() {
    grep -A2 ">$1<" "$metadata" |
        grep '<version' |
        sed -E 's/.*epoch="([^"]+)" ver="([^"]+)" rel="([^"]+)".*/\1:\2-\3/' |
        sort -V | tail -1 || true
}

read_pinned() {
    sed -n "s/^[[:space:]]*${1}=\${${1}:-\(.*\)}/\1/p" "$RPM_DEPS" |
        grep '\.el9' | head -1 || true
}

declare -A pkg_map=(
    [LIBVIRT_VERSION]="libvirt-daemon-driver-qemu"
    [QEMU_VERSION]="qemu-kvm-core"
    [SEABIOS_VERSION]="seabios-bin"
    [EDK2_VERSION]="edk2-ovmf"
    [PASST_VERSION]="passt"
    [VIRTIOFSD_VERSION]="virtiofsd"
    [SWTPM_VERSION]="swtpm-tools"
    [LIBNBD_VERSION]="libnbd"
)

changes=0

for varname in "${!pkg_map[@]}"; do
    pkg="${pkg_map[$varname]}"
    current=$(read_pinned "$varname")
    latest=$(query_latest "$pkg")

    if [[ -z "$latest" ]]; then
        echo "SKIP $varname ($pkg not in repo)"
        continue
    fi
    if [[ -z "$current" ]]; then
        echo "SKIP $varname (not pinned in rpm-deps.sh)"
        continue
    fi
    [[ "$current" != "$latest" ]] || continue

    echo "$varname: $current -> $latest"
    changes=$((changes + 1))

    if $dry_run || $check_only; then
        continue
    fi
    sed -i "s|${varname}=\${${varname}:-${current}}|${varname}=\${${varname}:-${latest}}|" "$RPM_DEPS"
done

if [[ $changes -eq 0 ]]; then
    echo "All versions up to date."
    exit 0
fi

$check_only && { echo "$changes outdated"; exit 1; }
$dry_run && { echo "$changes would be updated"; exit 0; }
echo "Updated $changes version(s) in $RPM_DEPS"
