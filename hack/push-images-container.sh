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
# Copyright 2026 The KubeVirt Authors.
#

set -e

source hack/common.sh
source hack/config.sh
source hack/container-utils.sh

KUBEVIRT_CRI=${KUBEVIRT_CRI:-$(determine_cri_bin)}
echo "Using container engine: ${KUBEVIRT_CRI}"

DOCKER_TAG=${DOCKER_TAG:-latest}

# Prefer DOCKER_PREFIX env var, then fall back to docker_prefix,
# then use default quay.io/kubevirt
if [ -z "${DOCKER_PREFIX}" ]; then
    if [ -n "${docker_prefix}" ]; then
        DOCKER_PREFIX="${docker_prefix}"
        echo "Using registry from provider config: ${DOCKER_PREFIX}"
    else
        DOCKER_PREFIX="quay.io/kubevirt"
        echo "Using default registry: ${DOCKER_PREFIX}"
    fi
else
    echo "Using DOCKER_PREFIX from environment: ${DOCKER_PREFIX}"
fi

IMAGE_PREFIX=${IMAGE_PREFIX:-}
BUILD_ARCH=${BUILD_ARCH:-amd64}
DIGESTS_DIR=${DIGESTS_DIR:-${OUT_DIR}/digests}

default_targets="
    virt-operator
    virt-api
    virt-controller
    virt-handler
    virt-launcher
    virt-exportserver
    virt-exportproxy
    virt-synchronization-controller
    alpine-container-disk-demo
    fedora-with-test-tooling-container-disk
    vm-killer
    sidecar-shim
    disks-images-provider
"

# Add additional images for non-s390x architectures only
if [[ "${BUILD_ARCH}" != "s390x" ]]; then
    default_targets+="
        conformance
        pr-helper
        example-hook-sidecar
        example-disk-mutation-hook-sidecar
        example-cloudinit-hook-sidecar
        cirros-container-disk-demo
        cirros-custom-container-disk-demo
        virtio-container-disk
        alpine-ext-kernel-boot-demo
        alpine-with-test-tooling-container-disk
        fedora-realtime-container-disk
        network-slirp-binding
        network-passt-binding
        network-passt-binding-cni
    "
fi

# Allow override via PUSH_TARGETS
PUSH_TARGETS_ARRAY=(${PUSH_TARGETS:-${default_targets}})

build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)

PUSH_FAILED=()
PUSH_SUCCESS=()

echo "==============================================="
echo "Pushing KubeVirt Images with ${KUBEVIRT_CRI}"
echo "==============================================="
echo "Registry: ${DOCKER_PREFIX}"
echo "Tag: ${DOCKER_TAG}"
echo "Architecture: ${BUILD_ARCH}"
echo "Targets: ${PUSH_TARGETS_ARRAY[@]}"
if [ "$build_count" -gt 1 ]; then
    echo "Multi-arch build: YES (will add arch suffix to tags)"
else
    echo "Multi-arch build: NO (single tag)"
fi
echo "==============================================="

check_image_exists() {
    local full_tag=$1

    if ${KUBEVIRT_CRI} image exists "${full_tag}" 2>/dev/null; then
        return 0
    elif ${KUBEVIRT_CRI} image inspect "${full_tag}" &>/dev/null; then
        return 0
    else
        return 1
    fi
}

echo "Checking if images exist locally..."
MISSING_IMAGES=()

for image in ${PUSH_TARGETS_ARRAY[@]}; do
    # Determine the tag to check
    if [ "$build_count" -gt 1 ]; then
        for arch in ${BUILD_ARCH//,/ }; do
            ARCH_TAG=$(format_archname ${arch} tag)
            full_tag="${DOCKER_PREFIX}/${IMAGE_PREFIX}${image}:${DOCKER_TAG}-${ARCH_TAG}"

            if ! check_image_exists "${full_tag}"; then
                MISSING_IMAGES+=("${full_tag}")
            else
                echo "- ${image}:${DOCKER_TAG}-${ARCH_TAG}"
            fi
        done
    else
        full_tag="${DOCKER_PREFIX}/${IMAGE_PREFIX}${image}:${DOCKER_TAG}"

        if ! check_image_exists "${full_tag}"; then
            MISSING_IMAGES+=("${full_tag}")
        else
            echo "- ${image}:${DOCKER_TAG}"
        fi
    fi
done

if [ ${#MISSING_IMAGES[@]} -gt 0 ]; then
    echo "ERROR: The following images were not found locally:"
    for img in "${MISSING_IMAGES[@]}"; do
        echo "- ${img}"
    done
    echo "Please build images first"
    exit 1
fi

push_image() {
    local image_name=$1
    local arch=$2

    # Determine tag
    if [ "$build_count" -gt 1 ]; then
        local arch_tag=$(format_archname ${arch} tag)
        local full_tag="${DOCKER_PREFIX}/${IMAGE_PREFIX}${image_name}:${DOCKER_TAG}-${arch_tag}"
    else
        local full_tag="${DOCKER_PREFIX}/${IMAGE_PREFIX}${image_name}:${DOCKER_TAG}"
    fi

    echo "Pushing ${image_name}"
    echo "Tag: ${full_tag}"

    local push_flags=""
    if [[ "${full_tag}" == localhost:* ]] || [[ "${full_tag}" == registry:* ]]; then
        # Local registry - disable TLS verification
        if [[ "${KUBEVIRT_CRI}" == "podman" ]]; then
            push_flags="--tls-verify=false"
        elif [[ "${KUBEVIRT_CRI}" == "docker" ]]; then
            # Docker doesn't support --tls-verify, it uses daemon config
            push_flags=""
        fi
        echo "Using insecure registry flags: ${push_flags}"
    fi

    if ${KUBEVIRT_CRI} push ${push_flags} "${full_tag}"; then
        echo "Successfully pushed"

        # Create digest marker file
        local digest_dir="${DIGESTS_DIR}/${arch}/${image_name}"
        mkdir -p "${digest_dir}"
        touch "${digest_dir}/${image_name}.image"
        echo "- Created digest marker: ${digest_dir}/${image_name}.image"

        PUSH_SUCCESS+=("${image_name}:${full_tag}")
        return 0
    else
        echo "- Failed to push"
        PUSH_FAILED+=("${image_name}:${full_tag}")
        return 1
    fi
}

echo "Pushing images..."

if [ "$build_count" -gt 1 ]; then
    for arch in ${BUILD_ARCH//,/ }; do
        arch=$(format_archname ${arch})
        echo "=========================================="
        echo "Architecture: ${arch}"
        echo "=========================================="
        for image in ${PUSH_TARGETS_ARRAY[@]}; do
            push_image "${image}" "${arch}"
            echo ""
        done
    done
else
    arch=$(format_archname ${BUILD_ARCH})
    for image in ${PUSH_TARGETS_ARRAY[@]}; do
        push_image "${image}" "${arch}"
        echo ""
    done
fi

echo "==============================================="
echo "Push Summary"
echo "==============================================="
echo "Successfully pushed: ${#PUSH_SUCCESS[@]} images"
for img in "${PUSH_SUCCESS[@]}"; do
    echo "- ${img}"
done
