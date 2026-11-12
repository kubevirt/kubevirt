#!/usr/bin/env bash
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
# Copyright 2025 Red Hat, Inc.
#
# Generates RPM rootfs tars for base images using standalone bazeldnf.
#
# Instead of invoking `bazel build //rpm:<rpmtree>`, this script:
#   1. Parses the rpmtree() rule from rpm/BUILD.bazel to get RPM names
#   2. Looks up download URLs from WORKSPACE
#   3. Downloads the RPMs
#   4. Calls `bazeldnf rpm2tar` with the correct symlinks/capabilities
#
# Usage:
#   hack/rpm-base-images/generate-rpm-tars.sh <rpmtree_name> [<output_tar>]
#
# Examples:
#   hack/rpm-base-images/generate-rpm-tars.sh launcherbase_x86_64_cs9
#   hack/rpm-base-images/generate-rpm-tars.sh exportserverbase_x86_64_cs9 _out/rpm-tars/exportserverbase_x86_64_cs9.tar
#
# Or source it and call generate_rpm_tar directly:
#   source hack/rpm-base-images/generate-rpm-tars.sh
#   generate_rpm_tar launcherbase_x86_64_cs9 _out/rpm-tars/launcherbase_x86_64_cs9.tar

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

BUILDFILE="${REPO_ROOT}/rpm/BUILD.bazel"
WORKSPACE_FILE="${REPO_ROOT}/WORKSPACE"
OUTPUT_DIR="${REPO_ROOT}/_out/rpm-tars"

source "${REPO_ROOT}/hack/install-bazeldnf.sh"

# Extract the list of RPM package names from a rpmtree() rule in BUILD.bazel.
# Each RPM reference looks like: "@acl-0__2.3.1-4.el9.x86_64//rpm",
extract_rpm_names() {
    local rpmtree_name="$1"
    sed -n "/name = \"${rpmtree_name}\"/,/^)/{
        s/.*\"@\(.*\)\/\/rpm\".*/\1/p
    }" "${BUILDFILE}"
}

# Extract symlinks from a rpmtree() rule.
# Returns lines of key=value pairs suitable for --symlinks flag.
# Parses blocks like:
#   symlinks = {
#       "/var/run": "../run",
#       "/usr/bin/nc": "/usr/bin/ncat",
#   },
extract_symlinks() {
    local rpmtree_name="$1"
    sed -n "/name = \"${rpmtree_name}\"/,/^)/{
        /symlinks = {/,/}/{ 
            /\".*\":.*\"/{ 
                s/.*\"\([^\"]*\)\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1=\2/p
            }
        }
    }" "${BUILDFILE}"
}

# Extract capabilities from a rpmtree() rule.
# Returns lines of path=capability pairs suitable for --capabilities flag.
# Parses blocks like:
#   capabilities = {
#       "/usr/libexec/qemu-kvm": [
#           "cap_net_bind_service",
#       ],
#   },
extract_capabilities() {
    local rpmtree_name="$1"
    local current_path=""
    sed -n "/name = \"${rpmtree_name}\"/,/^)/{
        /capabilities = {/,/^[[:space:]]*}/p
    }" "${BUILDFILE}" | while IFS= read -r line; do
        if echo "${line}" | grep -q '^\s*"/' ; then
            current_path=$(echo "${line}" | sed 's/.*"\([^"]*\)".*/\1/')
        elif [ -n "${current_path}" ] && echo "${line}" | grep -q '"cap_' ; then
            local cap
            cap=$(echo "${line}" | sed 's/.*"\([^"]*\)".*/\1/')
            echo "${current_path}=${cap}"
            current_path=""
        fi
    done
}

# Look up the download URL for an RPM from WORKSPACE.
# rpm() entries look like:
#   rpm(
#       name = "acl-0__2.3.1-4.el9.x86_64",
#       ...
#       urls = [ "http://...", ... ],
#   )
lookup_rpm_url() {
    local rpm_name="$1"
    sed -n "/name = \"${rpm_name}\"/,/^)/{
        /urls/,/\]/{
            s/.*\"\(http[^\"]*\)\".*/\1/p
        }
    }" "${WORKSPACE_FILE}" | head -1
}

# Generate an RPM rootfs tar for the given rpmtree target.
#
# Arguments:
#   $1 - rpmtree name (e.g. "launcherbase_x86_64_cs9")
#   $2 - output tar path (optional, defaults to _out/rpm-tars/<name>.tar)
generate_rpm_tar() {
    local rpmtree_name="$1"
    local output_tar="${2:-${OUTPUT_DIR}/${rpmtree_name}.tar}"

    echo "==> Generating RPM tar for ${rpmtree_name}"

    local rpm_names
    rpm_names=$(extract_rpm_names "${rpmtree_name}")

    if [ -z "${rpm_names}" ]; then
        echo "ERROR: no RPMs found for rpmtree '${rpmtree_name}' in ${BUILDFILE}" >&2
        return 1
    fi

    local rpm_count
    rpm_count=$(echo "${rpm_names}" | wc -l | tr -d ' ')
    echo "    Found ${rpm_count} RPMs in rpmtree rule"

    local tmpdir
    tmpdir=$(mktemp -d)
    trap "rm -rf ${tmpdir}" RETURN

    local rpm_inputs=()
    local download_failed=false
    for rpm_name in ${rpm_names}; do
        local url
        url=$(lookup_rpm_url "${rpm_name}")

        if [ -z "${url}" ]; then
            echo "ERROR: could not find URL for RPM '${rpm_name}' in ${WORKSPACE_FILE}" >&2
            download_failed=true
            continue
        fi

        local rpm_file="${tmpdir}/${rpm_name}.rpm"
        if ! curl -sSL -o "${rpm_file}" "${url}"; then
            echo "ERROR: failed to download ${url}" >&2
            download_failed=true
            continue
        fi
        rpm_inputs+=("-i" "${rpm_file}")
    done

    if [ "${download_failed}" = true ]; then
        echo "ERROR: some RPM downloads failed, aborting" >&2
        return 1
    fi

    # Build the bazeldnf rpm2tar flags
    local rpm2tar_args=()

    local symlinks
    symlinks=$(extract_symlinks "${rpmtree_name}")
    if [ -n "${symlinks}" ]; then
        while IFS= read -r symlink; do
            rpm2tar_args+=("--symlinks" "${symlink}")
        done <<< "${symlinks}"
    fi

    local capabilities
    capabilities=$(extract_capabilities "${rpmtree_name}")
    if [ -n "${capabilities}" ]; then
        while IFS= read -r cap; do
            rpm2tar_args+=("--capabilities" "${cap}")
        done <<< "${capabilities}"
    fi

    mkdir -p "$(dirname "${output_tar}")"

    echo "    Running bazeldnf rpm2tar with ${#rpm_inputs[@]} inputs..."
    bazeldnf rpm2tar \
        "${rpm_inputs[@]}" \
        "${rpm2tar_args[@]}" \
        -o "${output_tar}"

    local tar_size
    tar_size=$(du -h "${output_tar}" | cut -f1)
    echo "    Generated ${output_tar} (${tar_size})"
}

# Download the libguestfs appliance archive and repack it as a tar
# suitable for the libguestfs-tools Containerfile.
# The appliance is an http_archive in WORKSPACE (not an rpmtree).
generate_appliance_tar() {
    local arch="$1"  # x86_64 or s390x
    local output_tar="${2:-${OUTPUT_DIR}/appliance_layer_${arch}.tar}"

    local ws_name="libguestfs-appliance-${arch}"
    local url
    url=$(sed -n "/name = \"${ws_name}\"/,/^)/{
        /urls/,/\]/{
            s/.*\"\(http[^\"]*\)\".*/\1/p
        }
    }" "${WORKSPACE_FILE}" | head -1)

    if [ -z "${url}" ]; then
        echo "WARNING: no appliance URL found for ${ws_name} in WORKSPACE, skipping" >&2
        return 1
    fi

    echo "==> Downloading libguestfs appliance for ${arch}"

    local tmpdir
    tmpdir=$(mktemp -d)
    trap "rm -rf ${tmpdir}" RETURN

    local archive="${tmpdir}/appliance.tar.xz"
    curl -sSL -o "${archive}" "${url}"

    mkdir -p "${tmpdir}/appliance" "$(dirname "${output_tar}")"
    tar -xf "${archive}" -C "${tmpdir}/appliance"

    # Repack as a tar with the target directory structure
    # matching what Bazel's pkg_tar produces: /usr/local/lib/guestfs/appliance/
    mkdir -p "${tmpdir}/layer/usr/local/lib/guestfs/appliance"
    cp "${tmpdir}/appliance/appliance/"* "${tmpdir}/layer/usr/local/lib/guestfs/appliance/" 2>/dev/null || true

    # Create a "done" marker file (matches Bazel's done-file)
    touch "${tmpdir}/layer/usr/local/lib/guestfs/appliance/done"

    tar -cf "${output_tar}" -C "${tmpdir}/layer" .

    local tar_size
    tar_size=$(du -h "${output_tar}" | cut -f1)
    echo "    Generated ${output_tar} (${tar_size})"
}

# When invoked directly (not sourced), generate the tar for the given target
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [ $# -lt 1 ]; then
        echo "Usage: $0 <rpmtree_name> [<output_tar>]" >&2
        echo "" >&2
        echo "Examples:" >&2
        echo "  $0 launcherbase_x86_64_cs9" >&2
        echo "  $0 exportserverbase_x86_64_cs9 _out/rpm-tars/exportserverbase.tar" >&2
        exit 1
    fi

    generate_rpm_tar "$@"
fi
