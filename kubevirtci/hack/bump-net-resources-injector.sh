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
# Copyright The KubeVirt Authors.

set -eo pipefail

REPO="k8snetworkplumbingwg/network-resources-injector"
RAW_URL="https://raw.githubusercontent.com/${REPO}"
IMAGE="ghcr.io/k8snetworkplumbingwg/network-resources-injector"

MANIFEST_DIRS=(
    "./cluster-provision/gocli/opts/network_resources_injector/manifests"
    "./cluster-up/cluster/kind-sriov/sriov-components/manifests/network_resources_injector"
)

function usage() {
    echo "Update Network Resources Injector manifests"
    echo ""
    echo "Usage: $0 [VERSION|--check|--help]"
    echo ""
    echo "  (no args)   Update to latest version from GitHub"
    echo "  VERSION     Update to specific version (e.g., v1.8.0)"
    echo "  --check     Show current version"
    echo "  --help      Show this help message"
    echo ""
    echo "Upstream: https://github.com/${REPO}"
}

function get_latest_version() {
    curl -fsSL -H 'Accept: application/json' "https://github.com/${REPO}/releases/latest" | jq -r .tag_name
}

function get_current_version() {
    grep -oP 'network-resources-injector:\K[^\s"]+' "${MANIFEST_DIRS[0]}/server.yaml" | head -1
}

function apply_customizations() {
    local file="$1"
    local version="$2"

    # Replace image with full registry path and pinned version
    grep -q "image: network-resources-injector:latest" "${file}" || \
        { echo "ERROR: Pattern 'image: network-resources-injector:latest' not found in ${file}"; exit 1; }
    sed -i "s|image: network-resources-injector:latest|image: ${IMAGE}:${version}|g" "${file}"

    # Change imagePullPolicy from IfNotPresent to Always
    grep -q "imagePullPolicy: IfNotPresent" "${file}" || \
        { echo "ERROR: Pattern 'imagePullPolicy: IfNotPresent' not found in ${file}"; exit 1; }
    sed -i "s|imagePullPolicy: IfNotPresent|imagePullPolicy: Always|g" "${file}"

    # Add extra args after -logtostderr line
    grep -q "\-logtostderr$" "${file}" || \
        { echo "ERROR: Pattern '-logtostderr' not found in ${file}"; exit 1; }
    sed -i '/-logtostderr$/a\        - -v=2\n        - -insecure' "${file}"
}

function main() {
    if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
        usage
        exit 0
    fi

    if [[ "${1:-}" == "--check" ]]; then
        echo "Current version: $(get_current_version)"
        exit 0
    fi

    local version="${1:-}"
    if [[ -z "${version}" ]]; then
        version=$(get_latest_version)
        echo "Latest version: ${version}"
    fi

    local current_version
    current_version=$(get_current_version)
    echo "Updating Network Resources Injector from ${current_version} to ${version}..."

    # Fetch and customize manifests
    for manifest in auth.yaml server.yaml service.yaml; do
        url="${RAW_URL}/${version}/deployments/${manifest}"
        if ! content=$(curl -fsSL "${url}"); then
            echo "ERROR: Failed to fetch ${url}"
            exit 1
        fi

        for dir in "${MANIFEST_DIRS[@]}"; do
            echo "${content}" > "${dir}/${manifest}"

            # Apply customizations only to server.yaml
            if [[ "${manifest}" == "server.yaml" ]]; then
                apply_customizations "${dir}/${manifest}" "${version}"
            fi
        done
        echo "  Updated ${manifest}"
    done

    # Update pre-pull-images for all k8s providers
    echo "Updating pre-pull-images..."
    for k8s_provider in $(cd ./cluster-provision/k8s && ls -d [0-9].[0-9][0-9] 2>/dev/null); do
        ./cluster-provision/k8s/update-pre-pull-images.sh "${k8s_provider}"
    done

    echo "Done. Network Resources Injector updated to ${version}"
}

main "$@"
