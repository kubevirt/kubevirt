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
# Copyright 2026 The KubeVirt Authors
#
# Push pre-built RPM base images to the registry and update
# the BASE_IMAGE_VERSIONS file with their digests.

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

export KUBEVIRTCI_PATH=${KUBEVIRTCI_PATH:-"${REPO_ROOT}/kubevirtci/"}
export KUBEVIRTCI_CONFIG_PATH=${KUBEVIRTCI_CONFIG_PATH:-"${REPO_ROOT}/_ci-configs"}

source "${REPO_ROOT}/hack/common.sh"
source "${REPO_ROOT}/hack/container-utils.sh"

KUBEVIRT_CRI=${KUBEVIRT_CRI:-$(determine_cri_bin)}

PLATFORM=$(uname -m)
case ${PLATFORM} in
x86_64* | i?86_64* | amd64*) BUILD_ARCH=${BUILD_ARCH:-amd64} ;;
aarch64* | arm64*) BUILD_ARCH=${BUILD_ARCH:-arm64} ;;
s390x) BUILD_ARCH=${BUILD_ARCH:-s390x} ;;
*)
    echo "Unsupported architecture: ${PLATFORM}"
    exit 1
    ;;
esac

DOCKER_PREFIX=${DOCKER_PREFIX:-quay.io/vamsi_siddu}
BASE_IMAGE_TAG=${BASE_IMAGE_TAG:-latest}

VERSIONS_FILE="${SCRIPT_DIR}/BASE_IMAGE_VERSIONS"

build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)

get_base_images_for_arch() {
    local arch=$1
    local arch_tag=$(format_archname ${arch} tag)
    local images=(
        "launcherbase"
        "libvirt-devel"
        "handlerbase"
        "exportserverbase"
        "sidecar-shim"
        "testimage"
    )

    if [[ "${arch_tag}" != "s390x" ]]; then
        images+=("pr-helper")
    fi

    if [[ "${arch_tag}" != "arm64" ]]; then
        images+=("libguestfs-tools")
    fi

    if [[ -n "${BASE_IMAGE_TARGETS:-}" ]]; then
        IFS=',' read -ra images <<<"${BASE_IMAGE_TARGETS}"
    fi

    echo "${images[@]}"
}

echo "==============================================="
echo "Pushing RPM Base Images"
echo "==============================================="
echo "Registry: ${DOCKER_PREFIX}"
echo "Tag: ${BASE_IMAGE_TAG}"
echo "Architecture(s): ${BUILD_ARCH}"
echo "Multi-arch: $([ "$build_count" -gt 1 ] && echo "YES (will create manifests)" || echo "NO")"
echo "==============================================="

PUSH_SUCCESS=()
PUSH_FAILED=()

get_push_flags() {
    local tag=$1
    local flags=""
    if [[ "${tag}" == localhost:* ]] || [[ "${tag}" == registry:* ]]; then
        if [[ "${KUBEVIRT_CRI}" == "podman" ]]; then
            flags="--tls-verify=false"
        fi
    fi
    echo "${flags}"
}

push_single_image() {
    local full_tag=$1
    local push_flags=$(get_push_flags "${full_tag}")

    echo "Pushing ${full_tag}..."
    if ${KUBEVIRT_CRI} push ${push_flags} "${full_tag}"; then
        echo "Successfully pushed ${full_tag}"
        return 0
    else
        echo "ERROR: Failed to push ${full_tag}"
        return 1
    fi
}

if [ "$build_count" -gt 1 ]; then
    # Multi-arch: push each arch-tagged image, then create manifests
    for arch in ${BUILD_ARCH//,/ }; do
        arch_tag=$(format_archname ${arch} tag)
        local_images=()
        read -ra local_images <<<"$(get_base_images_for_arch ${arch})"

        echo ""
        echo "=========================================="
        echo "Pushing images for architecture: ${arch_tag}"
        echo "=========================================="

        for image in "${local_images[@]}"; do
            full_tag="${DOCKER_PREFIX}/${image}:${BASE_IMAGE_TAG}-${arch_tag}"
            if ! push_single_image "${full_tag}"; then
                PUSH_FAILED+=("${image}:${arch_tag}")
            fi
        done
    done

    echo ""
    echo "=========================================="
    echo "Creating multi-arch manifests"
    echo "=========================================="

    # Create manifests for images common to all architectures
    all_common_images=(
        "launcherbase"
        "libvirt-devel"
        "handlerbase"
        "exportserverbase"
        "sidecar-shim"
        "testimage"
    )

    # Add arch-specific images with only the arches that have them
    for image in "${all_common_images[@]}" "pr-helper" "libguestfs-tools"; do
        manifest_tag="${DOCKER_PREFIX}/${image}:${BASE_IMAGE_TAG}"
        echo ""
        echo "Creating manifest: ${manifest_tag}"

        arch_tags=()
        for arch in ${BUILD_ARCH//,/ }; do
            arch_tag=$(format_archname ${arch} tag)
            # Only include if this image exists for this arch
            local_images=()
            read -ra local_images <<<"$(get_base_images_for_arch ${arch})"
            for img in "${local_images[@]}"; do
                if [[ "${img}" == "${image}" ]]; then
                    arch_tags+=("${DOCKER_PREFIX}/${image}:${BASE_IMAGE_TAG}-${arch_tag}")
                    break
                fi
            done
        done

        if [ ${#arch_tags[@]} -eq 0 ]; then
            echo "Skipping ${image} (not available for any requested architecture)"
            continue
        fi

        if [[ "${KUBEVIRT_CRI}" == "podman" ]]; then
            ${KUBEVIRT_CRI} manifest rm "${manifest_tag}" 2>/dev/null || true
            ${KUBEVIRT_CRI} rmi "${manifest_tag}" 2>/dev/null || true
            ${KUBEVIRT_CRI} manifest create "${manifest_tag}" "${arch_tags[@]}"
            push_flags=$(get_push_flags "${manifest_tag}")
            ${KUBEVIRT_CRI} manifest push ${push_flags} "${manifest_tag}" "docker://${manifest_tag}"
        elif [[ "${KUBEVIRT_CRI}" == "docker" ]]; then
            docker buildx imagetools create -t "${manifest_tag}" "${arch_tags[@]}"
        fi

        PUSH_SUCCESS+=("${image}")
        echo "Created and pushed manifest: ${manifest_tag}"
    done
else
    # Single-arch: push directly
    local_images=()
    read -ra local_images <<<"$(get_base_images_for_arch ${BUILD_ARCH})"

    for image in "${local_images[@]}"; do
        full_tag="${DOCKER_PREFIX}/${image}:${BASE_IMAGE_TAG}"
        if push_single_image "${full_tag}"; then
            PUSH_SUCCESS+=("${image}")
        else
            PUSH_FAILED+=("${image}")
        fi
    done
fi

if [[ ${#PUSH_FAILED[@]} -gt 0 ]]; then
    echo ""
    echo "ERROR: Failed to push: ${PUSH_FAILED[*]}"
    exit 1
fi

echo ""
echo "==============================================="
echo "Updating ${VERSIONS_FILE}"
echo "==============================================="

{
    echo "# RPM Base Image Versions"
    echo "# Auto-generated by push-base-images.sh"
    echo "# Update this file by running: hack/rpm-base-images/push-base-images.sh"
    echo "#"
    echo "# Format: IMAGE_NAME=REGISTRY/IMAGE:TAG"
    echo ""
    echo "DOCKER_PREFIX=${DOCKER_PREFIX}"
    echo "BASE_IMAGE_TAG=${BASE_IMAGE_TAG}"
    echo ""
    for image in "${PUSH_SUCCESS[@]}"; do
        echo "${image}=${DOCKER_PREFIX}/${image}:${BASE_IMAGE_TAG}"
    done
} >"${VERSIONS_FILE}"

echo "Updated ${VERSIONS_FILE}"
echo ""
echo "==============================================="
echo "Push completed successfully!"
echo "==============================================="
