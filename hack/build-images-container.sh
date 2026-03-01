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

set -e

source hack/common.sh
source hack/config.sh
source hack/version.sh
source hack/container-utils.sh

KUBEVIRT_CRI=${KUBEVIRT_CRI:-$(determine_cri_bin)}
echo "Using container engine: ${KUBEVIRT_CRI}"

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

DOCKER_TAG=${DOCKER_TAG:-devel}

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
BUILD_OUTPUT_DIR=${BUILD_OUTPUT_DIR:-_out}
DIGESTS_DIR=${DIGESTS_DIR:-${BUILD_OUTPUT_DIR}/digests}

# Builder image configuration
BUILDER_VERSION=$(grep 'kubevirt_builder_version=' hack/dockerized | cut -d'"' -f2)
BUILDER_IMAGE=${BUILDER_IMAGE:-quay.io/kubevirt/builder:${BUILDER_VERSION}}

PLATFORM_ARCH=$(format_archname ${BUILD_ARCH} tag)
BAZEL_ARCH=$(format_archname ${BUILD_ARCH})

# Distroless base image digests per architecture - matches WORKSPACE
case ${PLATFORM_ARCH} in
amd64)
    DISTROLESS_DIGEST="sha256:0ba6aa6b538aeae3d0f716ea8837703eb147173cd673241662e89adb794da829"
    ;;
arm64)
    DISTROLESS_DIGEST="sha256:9ee08ca352647dad1511153afb18f4a6dbb4f56bafc7d618d0082c16a14cfdf1"
    ;;
s390x)
    DISTROLESS_DIGEST="sha256:6e2e356c462d69668a0313bf45ed3de614e9d4e0b9c03fa081d3bcae143a58ba"
    ;;
*)
    echo "Error: Unsupported architecture ${PLATFORM_ARCH} (from BUILD_ARCH=${BUILD_ARCH})"
    exit 1
    ;;
esac

DISTROLESS_BASE_IMAGE="gcr.io/distroless/base-debian12@${DISTROLESS_DIGEST}"

kubevirt::version::get_version_vars
KUBEVIRT_VERSION=${KUBEVIRT_GIT_VERSION:-"devel"}
CENTOS_STREAM_VERSION=${KUBEVIRT_CENTOS_STREAM_VERSION:-9}
BUILD_ARGS="--build-arg KUBEVIRT_VERSION=${KUBEVIRT_VERSION}"
BUILD_ARGS+=" --build-arg BUILD_ARCH=${BUILD_ARCH}"
BUILD_ARGS+=" --build-arg BAZEL_ARCH=${BAZEL_ARCH}"
BUILD_ARGS+=" --build-arg BUILDER_IMAGE=${BUILDER_IMAGE}"
BUILD_ARGS+=" --build-arg DISTROLESS_BASE_IMAGE=${DISTROLESS_BASE_IMAGE}"
BUILD_ARGS+=" --build-arg CENTOS_STREAM_VERSION=${CENTOS_STREAM_VERSION}"

default_targets="
    virt-operator
    virt-api
    virt-controller
    virt-handler
    virt-launcher
    virt-exportserver
    virt-exportproxy
    virt-synchronization-controller
    virt-template-apiserver
    virt-template-controller
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

# Add libguestfs-tools for x86_64 and s390x only (not arm64)
if [[ "${BUILD_ARCH}" != "arm64" && "${BUILD_ARCH}" != "aarch64" && "${BUILD_ARCH}" != "crossbuild-aarch64" ]]; then
    default_targets+="
        libguestfs-tools
    "
fi

# Allow override via PUSH_TARGETS
BUILD_TARGETS=(${PUSH_TARGETS:-${default_targets}})

get_containerfile_path() {
    local image_name=$1

    # Map image names to their Containerfile locations
    case "$image_name" in
    virt-operator | virt-api | virt-controller | virt-handler | virt-launcher | virt-exportserver | virt-exportproxy)
        echo "cmd/${image_name}/Containerfile"
        ;;
    virt-synchronization-controller)
        echo "cmd/synchronization-controller/Containerfile"
        ;;
    pr-helper)
        echo "cmd/${image_name}/Containerfile"
        ;;
    vm-killer)
        echo "images/vm-killer/Containerfile"
        ;;
    kubevirt-testing-base)
        echo "images/kubevirt-testing-base/Containerfile"
        ;;
    disks-images-provider)
        echo "images/disks-images-provider/Containerfile"
        ;;
    winrmcli)
        echo "images/winrmcli/Containerfile"
        ;;
    conformance)
        echo "tests/conformance/Containerfile"
        ;;
    libguestfs-tools)
        echo "cmd/libguestfs/Containerfile"
        ;;
    sidecar-shim)
        echo "cmd/sidecars/Containerfile"
        ;;
    network-slirp-binding)
        echo "cmd/sidecars/network-slirp-binding/Containerfile"
        ;;
    network-passt-binding)
        echo "cmd/sidecars/network-passt-binding/Containerfile"
        ;;
    network-passt-binding-cni)
        echo "cmd/cniplugins/passt-binding/cmd/Containerfile"
        ;;
    example-hook-sidecar)
        echo "cmd/sidecars/smbios/Containerfile"
        ;;
    example-disk-mutation-hook-sidecar)
        echo "cmd/sidecars/disk-mutation/Containerfile"
        ;;
    example-cloudinit-hook-sidecar)
        echo "cmd/sidecars/cloudinit/Containerfile"
        ;;
    *)
        echo "ERROR: Unknown image: $image_name" >&2
        return 1
        ;;
    esac
}

build_image() {
    local image_name=$1
    local containerfile=$2
    local context=${3:-.}
    local extra_build_args=${4:-}

    local full_tag="${DOCKER_PREFIX}/${IMAGE_PREFIX}${image_name}:${DOCKER_TAG}"

    echo "Building ${image_name} for ${BUILD_ARCH} (linux/${PLATFORM_ARCH})"

    local build_cmd="${KUBEVIRT_CRI} build ${BUILD_ARGS} --platform linux/${PLATFORM_ARCH}"

    if [[ -n "${extra_build_args}" ]]; then
        build_cmd+=" ${extra_build_args}"
    fi

    build_cmd+=" -f ${containerfile} -t ${full_tag} ${context}"

    eval ${build_cmd}

    save_image_digest "${image_name}" "${full_tag}" "${BUILD_ARCH}"

    echo "Successfully built ${full_tag}"
}

# Check if an image should be retagged instead of built
is_retagged_image() {
    local image_name=$1
    case "$image_name" in
    fedora-with-test-tooling-container-disk | alpine-with-test-tooling-container-disk | fedora-realtime-container-disk | alpine-ext-kernel-boot-demo | virt-template-apiserver | virt-template-controller)
        return 0
        ;;
    *)
        return 1
        ;;
    esac
}

# Check if an image is a container disk built by separate script
is_container_disk_image() {
    local image_name=$1
    case "$image_name" in
    alpine-container-disk-demo | cirros-container-disk-demo | cirros-custom-container-disk-demo | virtio-container-disk)
        return 0
        ;;
    *)
        return 1
        ;;
    esac
}

# Retag a single upstream image
retag_single_image() {
    local image_name=$1

    case "$image_name" in
    fedora-with-test-tooling-container-disk)
        local normalized_arch=$(format_archname ${BUILD_ARCH} tag)
        case ${normalized_arch} in
        amd64)
            DIGEST="sha256:897af945d1c58366086d5933ae4f341a5f1413b88e6c7f2b659436adc5d0f522"
            ;;
        arm64)
            DIGEST="sha256:3d5a2a95f7f9382dc6730073fe19a6b1bc668b424c362339c88c6a13dff2ef49"
            ;;
        s390x)
            DIGEST="sha256:3d9f468750d90845a81608ea13c85237ea295c6295c911a99dc5e0504c8bc05b"
            ;;
        *)
            echo "ERROR: Unsupported architecture ${normalized_arch} for fedora-with-test-tooling"
            exit 1
            ;;
        esac
        SOURCE_IMAGE="quay.io/kubevirtci/fedora-with-test-tooling@${DIGEST}"
        ;;
    alpine-with-test-tooling-container-disk)
        SOURCE_IMAGE="quay.io/kubevirtci/alpine-with-test-tooling-container-disk@sha256:882450d4a0141d29422d049390cae59176c1a9ddf75bd9f3ecdd5a9081c7d95b"
        ;;
    fedora-realtime-container-disk)
        SOURCE_IMAGE="quay.io/kubevirt/fedora-realtime-container-disk@sha256:f91379d202a5493aba9ce06870b5d1ada2c112f314530c9820a9ad07426aa565"
        ;;
    alpine-ext-kernel-boot-demo)
        SOURCE_IMAGE="quay.io/kubevirt/alpine-ext-kernel-boot-demo@sha256:bccd990554f55623d96fa70bc7efc553dd617523ebca76919b917ad3ee616c1d"
        ;;
    virt-template-apiserver)
        local normalized_arch=$(format_archname ${BUILD_ARCH} tag)
        case ${normalized_arch} in
        amd64)
            DIGEST="sha256:c3b20cd9bc83cc9065998b78cffe1c1cea323231ad1c5678aeefe82a5d172846"
            ;;
        arm64)
            DIGEST="sha256:dc6f6773fe84412f1a6cd6087523e44c6727d6e519f1223ea96e4d607aa90c85"
            ;;
        s390x)
            DIGEST="sha256:0a2891de175e312a6ec9f4a4670c11e713b3a0ead2ff8c6bad30ff72e71bcf7b"
            ;;
        *)
            echo "ERROR: Unsupported architecture ${normalized_arch} for virt-template-apiserver"
            exit 1
            ;;
        esac
        SOURCE_IMAGE="quay.io/kubevirt/virt-template-apiserver@${DIGEST}"
        ;;
    virt-template-controller)
        local normalized_arch=$(format_archname ${BUILD_ARCH} tag)
        case ${normalized_arch} in
        amd64)
            DIGEST="sha256:e68cf77970e57aaec88ead5da9862a83632c5ba35ee3e4511355b6fbd7d421a9"
            ;;
        arm64)
            DIGEST="sha256:54122356de3461714e3dbbf2942bb6182e48b294f89ba08e57472cfc777534c5"
            ;;
        s390x)
            DIGEST="sha256:c178ec45bf32c9553f6e962e020cdfed427e31d20a0c1fda9f8e148c58d39504"
            ;;
        *)
            echo "ERROR: Unsupported architecture ${normalized_arch} for virt-template-controller"
            exit 1
            ;;
        esac
        SOURCE_IMAGE="quay.io/kubevirt/virt-template-controller@${DIGEST}"
        ;;
    *)
        echo "ERROR: Unknown retagged image: $image_name" >&2
        return 1
        ;;
    esac

    local target_tag="${DOCKER_PREFIX}/${IMAGE_PREFIX}${image_name}:${DOCKER_TAG}"

    echo "${image_name}"
    echo "Source: ${SOURCE_IMAGE}"
    ${KUBEVIRT_CRI} pull ${SOURCE_IMAGE}
    ${KUBEVIRT_CRI} tag ${SOURCE_IMAGE} ${target_tag}
    save_image_digest "${image_name}" "${target_tag}" "${BUILD_ARCH}"
}

setup_buildah_context

echo "Creating .version file..."
./hack/create-version-file.sh .version

echo "Building KubeVirt Images with ${KUBEVIRT_CRI}"
echo "Architecture: ${BUILD_ARCH}"
echo "Version: ${KUBEVIRT_VERSION}"
echo "Tag: ${DOCKER_TAG}"
echo "Prefix: ${DOCKER_PREFIX}"
echo "Targets: ${BUILD_TARGETS[@]}"

# Separate container disk images from regular builds
CONTAINER_DISK_IMAGES=""
REGULAR_BUILD_IMAGES=""
NEEDS_TESTING_BASE=false
NEEDS_SIDECAR_SHIM=false

for image in ${BUILD_TARGETS[@]}; do
    if is_retagged_image "${image}"; then
        :
    elif is_container_disk_image "${image}"; then
        CONTAINER_DISK_IMAGES+=" ${image}"
    else
        REGULAR_BUILD_IMAGES+=" ${image}"

        # Check if this image needs kubevirt-testing-base
        if [[ "${image}" == "disks-images-provider" || "${image}" == "winrmcli" || "${image}" == "vm-killer" ]]; then
            NEEDS_TESTING_BASE=true
        fi

        # Check if this image needs sidecar-shim
        if [[ "${image}" == "example-hook-sidecar" || "${image}" == "example-cloudinit-hook-sidecar" || "${image}" == "example-disk-mutation-hook-sidecar" ]]; then
            NEEDS_SIDECAR_SHIM=true
        fi
    fi
done

# Build kubevirt-testing-base first if needed (only if NOT in main build targets)
if [[ "${NEEDS_TESTING_BASE}" == "true" ]]; then
    if [[ ! " ${BUILD_TARGETS[@]} " =~ " kubevirt-testing-base " ]]; then
        containerfile=$(get_containerfile_path "kubevirt-testing-base")
        if [ -f "${containerfile}" ]; then
            echo "Pre-building kubevirt-testing-base as dependency..."
            build_image "kubevirt-testing-base" "${containerfile}" "."
        else
            echo "ERROR: kubevirt-testing-base Containerfile not found but required"
            exit 1
        fi
    fi
fi

# Build sidecar-shim first if needed (only if NOT in main build targets)
if [[ "${NEEDS_SIDECAR_SHIM}" == "true" ]]; then
    if [[ ! " ${BUILD_TARGETS[@]} " =~ " sidecar-shim " ]]; then
        containerfile=$(get_containerfile_path "sidecar-shim")
        if [ -f "${containerfile}" ]; then
            echo "Pre-building sidecar-shim as dependency..."
            build_image "sidecar-shim" "${containerfile}" "."
        else
            echo "ERROR: sidecar-shim Containerfile not found but required"
            exit 1
        fi
    fi
fi

for image in ${BUILD_TARGETS[@]}; do
    if is_retagged_image "${image}"; then
        normalized_arch=$(format_archname ${BUILD_ARCH} tag)
        if [[ "${image}" == "fedora-realtime-container-disk" && "${normalized_arch}" != "amd64" ]]; then
            echo "==> Skipping ${image} (not available for ${BUILD_ARCH})"
            continue
        fi

        retag_single_image "${image}"
    fi
done

for image in ${REGULAR_BUILD_IMAGES}; do
    containerfile=$(get_containerfile_path "${image}")

    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to get Containerfile path for ${image}"
        exit 1
    fi

    if [ ! -f "${containerfile}" ]; then
        echo "ERROR: Containerfile not found for ${image}: ${containerfile}"
        echo "       This image needs to be implemented."
        exit 1
    fi

    extra_args=""
    if [[ "${image}" == "disks-images-provider" || "${image}" == "winrmcli" || "${image}" == "vm-killer" ]]; then
        extra_args="--build-arg TESTING_BASE_IMAGE=${DOCKER_PREFIX}/kubevirt-testing-base:${DOCKER_TAG}"
    elif [[ "${image}" == "example-hook-sidecar" || "${image}" == "example-cloudinit-hook-sidecar" || "${image}" == "example-disk-mutation-hook-sidecar" ]]; then
        extra_args="--build-arg SIDECAR_SHIM_IMAGE=${DOCKER_PREFIX}/sidecar-shim:${DOCKER_TAG}"
    fi

    build_image "${image}" \
        "${containerfile}" \
        "." \
        "${extra_args}"
done

if [[ -n "${CONTAINER_DISK_IMAGES}" ]]; then
    echo "Building container disk images via build-container-disks.sh"
    BUILD_ARCH=${PLATFORM_ARCH} \
        DOCKER_TAG=${DOCKER_TAG} \
        DOCKER_PREFIX=${DOCKER_PREFIX} \
        IMAGE_PREFIX=${IMAGE_PREFIX} \
        KUBEVIRT_CRI=${KUBEVIRT_CRI} \
        ./hack/build-container-disks.sh
fi

# Cleanup .version file that was created during the build
rm -f .version

echo "==============================================="
echo "Build completed successfully!"
echo "==============================================="
