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
# Build RPM base images for KubeVirt components.
# These images contain only the RPM-based filesystem layers and are
# rebuilt only when RPM dependencies change.

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
BUILDER_VERSION=$(grep '^kubevirt_builder_version=' "${REPO_ROOT}/hack/dockerized" | cut -d'"' -f2)
BUILDER_IMAGE=${BUILDER_IMAGE:-quay.io/kubevirt/builder:${BUILDER_VERSION}}
CENTOS_STREAM_VERSION=${CENTOS_STREAM_VERSION:-9}

build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)

echo "==============================================="
echo "Building RPM Base Images"
echo "==============================================="
echo "Architecture(s): ${BUILD_ARCH}"
echo "Multi-arch: $([ "$build_count" -gt 1 ] && echo "YES" || echo "NO")"
echo "Container Engine: ${KUBEVIRT_CRI}"
echo "Registry: ${DOCKER_PREFIX}"
echo "Tag: ${BASE_IMAGE_TAG}"
echo "Builder: ${BUILDER_IMAGE}"
echo "==============================================="

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

    # pr-helper not available for s390x
    if [[ "${arch_tag}" != "s390x" ]]; then
        images+=("pr-helper")
    fi

    # libguestfs-tools only for amd64 and s390x (not arm64)
    if [[ "${arch_tag}" != "arm64" ]]; then
        images+=("libguestfs-tools")
    fi

    # Allow override via BASE_IMAGE_TARGETS
    if [[ -n "${BASE_IMAGE_TARGETS:-}" ]]; then
        IFS=',' read -ra images <<<"${BASE_IMAGE_TARGETS}"
    fi

    echo "${images[@]}"
}

build_for_arch() {
    local arch=$1
    local arch_normalized=$(format_archname ${arch})
    local arch_tag=$(format_archname ${arch} tag)

    local -a arch_images
    read -ra arch_images <<<"$(get_base_images_for_arch ${arch})"

    echo "Targets for ${arch_tag}: ${arch_images[*]}"

    local arch_build_args="--build-arg BUILDER_IMAGE=${BUILDER_IMAGE}"
    arch_build_args+=" --build-arg BUILD_ARCH=${arch_normalized}"
    arch_build_args+=" --build-arg CENTOS_STREAM_VERSION=${CENTOS_STREAM_VERSION}"

    for image in "${arch_images[@]}"; do
        containerfile="${SCRIPT_DIR}/Containerfile.${image}"

        if [[ ! -f "${containerfile}" ]]; then
            echo "ERROR: Containerfile not found: ${containerfile}"
            exit 1
        fi

        # For multi-arch, append arch suffix to the tag
        if [ "$build_count" -gt 1 ]; then
            full_tag="${DOCKER_PREFIX}/${image}:${BASE_IMAGE_TAG}-${arch_tag}"
        else
            full_tag="${DOCKER_PREFIX}/${image}:${BASE_IMAGE_TAG}"
        fi

        echo ""
        echo "Building ${image} for ${arch_tag} -> ${full_tag}"
        ${KUBEVIRT_CRI} build \
            ${arch_build_args} \
            --platform "linux/${arch_tag}" \
            -f "${containerfile}" \
            -t "${full_tag}" \
            "${REPO_ROOT}"

        echo "Successfully built ${full_tag}"
    done
}

if [ "$build_count" -gt 1 ]; then
    for arch in ${BUILD_ARCH//,/ }; do
        echo ""
        echo "=========================================="
        echo "Building for architecture: ${arch}"
        echo "=========================================="
        build_for_arch "${arch}"
    done
else
    build_for_arch "${BUILD_ARCH}"
fi

echo ""
echo "==============================================="
echo "Base image build completed successfully!"
echo "==============================================="
