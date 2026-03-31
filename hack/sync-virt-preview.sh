#!/bin/bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright the KubeVirt Authors.
#

set -e

# Syncs the latest available builds of qemu and libvirt from the
# @virtmaint-sig/virt-preview COPR into the project's RPM dependencies
# by directly replacing rpm() entries in WORKSPACE and references in
# rpm/BUILD.bazel.
#
# This bypasses bazeldnf because the COPR packages are built from Fedora
# RPM specs and have a lower epoch than the CentOS Stream packages,
# causing bazeldnf to prefer the CentOS Stream versions.
#
# This is intended for local development and testing against newer virt
# stack builds. The resulting changes can be stashed in a DNM commit.
#
# Usage:
#   # Sync for CentOS Stream 9 (default)
#   hack/sync-virt-preview.sh
#
#   # Sync for CentOS Stream 10
#   KUBEVIRT_CENTOS_STREAM_VERSION=10 hack/sync-virt-preview.sh
#
#   # Sync for a single architecture
#   SINGLE_ARCH=x86_64 hack/sync-virt-preview.sh

KUBEVIRT_CENTOS_STREAM_VERSION=${KUBEVIRT_CENTOS_STREAM_VERSION:-9}
SINGLE_ARCH="${SINGLE_ARCH:-}"
COPR_BASE_URL="https://download.copr.fedorainfracloud.org/results/@virtmaint-sig/virt-preview"

cd "$(dirname "$0")/.."

ARCHS="${SINGLE_ARCH:-x86_64 aarch64 s390x}"
WORK=$(mktemp -d)
trap "rm -rf ${WORK}" EXIT

# Phase 1: Fetch COPR repodata and extract qemu-*/libvirt-* package metadata
#
# Output TSV: name, rpm_arch, epoch, ver, rel, sha256, href, ws_arch
COPR_PKGS="${WORK}/packages.tsv"
: > "${COPR_PKGS}"

for arch in ${ARCHS}; do
    repo_url="${COPR_BASE_URL}/centos-stream-${KUBEVIRT_CENTOS_STREAM_VERSION}-${arch}/"
    echo "Fetching COPR metadata for ${arch}..."

    repomd="${WORK}/repomd_${arch}.xml"
    if ! curl -sSfL "${repo_url}repodata/repomd.xml" -o "${repomd}" 2>/dev/null; then
        echo "  WARNING: failed to fetch repomd.xml for ${arch}, skipping"
        continue
    fi

    # Extract primary.xml href from repomd.xml
    primary_href=$(sed -n '/<data type="primary">/,/<\/data>/{
        /location.*href=/{
            s/.*href="//
            s/".*//
            p
        }
    }' "${repomd}")

    if [ -z "${primary_href}" ]; then
        echo "  WARNING: no primary metadata for ${arch}, skipping"
        continue
    fi

    # Fetch and decompress primary.xml
    primary="${WORK}/primary_${arch}.xml"
    case "${primary_href}" in
        *.gz)  curl -sSfL "${repo_url}${primary_href}" | gunzip > "${primary}" ;;
        *.xz)  curl -sSfL "${repo_url}${primary_href}" | xz -d > "${primary}" ;;
        *.zst) curl -sSfL "${repo_url}${primary_href}" | zstd -d > "${primary}" ;;
        *)     curl -sSfL "${repo_url}${primary_href}" -o "${primary}" ;;
    esac

    # Parse primary.xml for qemu-*/libvirt-* packages
    awk -v ws_arch="${arch}" '
        /<package type="rpm">/ {
            p=1; name=""; arch=""; epoch="0"; ver=""; rel=""; sha=""; href=""
        }
        /<\/package>/ {
            if (p && arch != "src" && (name ~ /^qemu-/ || name ~ /^libvirt-/))
                print name "\t" arch "\t" epoch "\t" ver "\t" rel "\t" sha "\t" href "\t" ws_arch
            p=0
        }
        p && /<name>/ {
            s=$0; sub(/.*<name>/, "", s); sub(/<\/name>.*/, "", s); name=s
        }
        p && /<arch>/ {
            s=$0; sub(/.*<arch>/, "", s); sub(/<\/arch>.*/, "", s); arch=s
        }
        p && /<version / {
            s=$0
            e=s; sub(/.*epoch="/, "", e); sub(/".*/, "", e); epoch=e
            v=s; sub(/.*ver="/, "", v); sub(/".*/, "", v); ver=v
            r=s; sub(/.*rel="/, "", r); sub(/".*/, "", r); rel=r
        }
        p && /<checksum type="sha256"/ {
            s=$0; sub(/.*<checksum[^>]*>/, "", s); sub(/<\/checksum>.*/, "", s); sha=s
        }
        p && /<location / {
            s=$0; sub(/.*href="/, "", s); sub(/".*/, "", s); href=s
        }
    ' "${primary}" >> "${COPR_PKGS}"

    count=$(awk 'END{print NR}' "${COPR_PKGS}")
    echo "  total packages so far: ${count}"
done

if [ ! -s "${COPR_PKGS}" ]; then
    echo "ERROR: no packages found in COPR"
    exit 1
fi

echo ""
echo "COPR packages found:"
awk -F'\t' '{ printf "  %-40s %s-%s (%s)\n", $1, $4, $5, $8 }' "${COPR_PKGS}"
echo ""

# Phase 2: Replace rpm() blocks in WORKSPACE
#
# For each existing qemu-*/libvirt-* rpm() entry matching the target CS
# version, look up the replacement in COPR data and rewrite the block.
# Record old→new name mappings for BUILD.bazel fixup.

MAPPINGS="${WORK}/mappings.tsv"
: > "${MAPPINGS}"

awk \
    -v cs_suffix=".el${KUBEVIRT_CENTOS_STREAM_VERSION}" \
    -v copr_base="${COPR_BASE_URL}" \
    -v cs_version="${KUBEVIRT_CENTOS_STREAM_VERSION}" \
    -v mappings_file="${MAPPINGS}" \
    '
FNR == NR {
    # Load COPR packages TSV (first file)
    split($0, f, "\t")
    key = f[1] SUBSEP f[8]  # name SUBSEP ws_arch
    copr_epoch[key] = f[3]
    copr_ver[key] = f[4]
    copr_rel[key] = f[5]
    copr_sha[key] = f[6]
    copr_href[key] = f[7]
    next
}

# Process WORKSPACE (second file)
/^rpm\($/ {
    in_rpm = 1
    block = $0 "\n"
    rpm_name = ""
    next
}

in_rpm {
    block = block $0 "\n"
    if ($0 ~ /name = "/) {
        s = $0
        sub(/.*name = "/, "", s)
        sub(/".*/, "", s)
        rpm_name = s
    }
    if ($0 ~ /^\)$/) {
        in_rpm = 0
        result = try_replace(rpm_name)
        if (result != "") {
            printf "%s", result
        } else {
            printf "%s", block
        }
    }
    next
}

{ print }

function try_replace(old_name,
    parts, nparts, left_parts, nleft, pkg_name, old_epoch, right, ws_arch, key, new_name, url, i) {

    # Split on "__"
    nparts = split(old_name, parts, "__")
    if (nparts != 2) return ""

    # Left side: "pkg-name-EPOCH" -> extract pkg_name and epoch
    nleft = split(parts[1], left_parts, "-")
    if (nleft < 2) return ""
    old_epoch = left_parts[nleft]
    pkg_name = left_parts[1]
    for (i = 2; i < nleft; i++) pkg_name = pkg_name "-" left_parts[i]

    # Check package name prefix
    if (pkg_name !~ /^qemu-/ && pkg_name !~ /^libvirt-/) return ""

    # Right side: "VER-REL.ARCH" -> extract arch
    right = parts[2]
    ws_arch = ""
    if (right ~ /\.x86_64$/) { ws_arch = "x86_64"; sub(/\.x86_64$/, "", right) }
    else if (right ~ /\.aarch64$/) { ws_arch = "aarch64"; sub(/\.aarch64$/, "", right) }
    else if (right ~ /\.s390x$/) { ws_arch = "s390x"; sub(/\.s390x$/, "", right) }
    else return ""

    # Only replace entries for the target CS version
    if (index(right, cs_suffix) == 0) return ""

    # Lookup in COPR data
    key = pkg_name SUBSEP ws_arch
    if (!(key in copr_epoch)) return ""

    new_name = pkg_name "-" copr_epoch[key] "__" copr_ver[key] "-" copr_rel[key] "." ws_arch
    if (new_name == old_name) return ""

    url = copr_base "/centos-stream-" cs_version "-" ws_arch "/" copr_href[key]

    # Record mapping for BUILD.bazel
    print old_name "\t" new_name >> mappings_file

    return "rpm(\n" \
        "    name = \"" new_name "\",\n" \
        "    sha256 = \"" copr_sha[key] "\",\n" \
        "    urls = [\n" \
        "        \"" url "\",\n" \
        "    ],\n" \
        ")\n"
}
' "${COPR_PKGS}" WORKSPACE > "${WORK}/WORKSPACE.new"

# Phase 3: Apply changes

if [ ! -s "${MAPPINGS}" ]; then
    echo "No replacements needed - COPR packages may match current versions"
    exit 0
fi

cp "${WORK}/WORKSPACE.new" WORKSPACE

# Build a sed script from the mappings to update BUILD.bazel references
SED_SCRIPT="${WORK}/build.sed"
: > "${SED_SCRIPT}"
while IFS=$'\t' read -r old_name new_name; do
    # Escape special sed characters in names (/ is not used in names so | is safe)
    echo "s|@${old_name}//rpm|@${new_name}//rpm|g" >> "${SED_SCRIPT}"
    echo "  REPLACE ${old_name}"
    echo "       -> ${new_name}"
done < "${MAPPINGS}"

sed -f "${SED_SCRIPT}" rpm/BUILD.bazel > "${WORK}/BUILD.bazel.new"
cp "${WORK}/BUILD.bazel.new" rpm/BUILD.bazel

count=$(wc -l < "${MAPPINGS}")
echo ""
echo "Replaced ${count} package(s) in WORKSPACE and rpm/BUILD.bazel"
