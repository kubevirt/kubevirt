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

# RPM base image configuration
BASE_IMAGE_PREFIX=${BASE_IMAGE_PREFIX:-quay.io/vamsi_siddu}
BASE_IMAGE_TAG=${BASE_IMAGE_TAG:-latest}
LAUNCHERBASE_IMAGE=${LAUNCHERBASE_IMAGE:-${BASE_IMAGE_PREFIX}/launcherbase:${BASE_IMAGE_TAG}}
LIBVIRT_DEVEL_IMAGE=${LIBVIRT_DEVEL_IMAGE:-${BASE_IMAGE_PREFIX}/libvirt-devel:${BASE_IMAGE_TAG}}
HANDLERBASE_IMAGE=${HANDLERBASE_IMAGE:-${BASE_IMAGE_PREFIX}/handlerbase:${BASE_IMAGE_TAG}}
EXPORTSERVERBASE_IMAGE=${EXPORTSERVERBASE_IMAGE:-${BASE_IMAGE_PREFIX}/exportserverbase:${BASE_IMAGE_TAG}}
LIBGUESTFS_BASE_IMAGE=${LIBGUESTFS_BASE_IMAGE:-${BASE_IMAGE_PREFIX}/libguestfs-tools:${BASE_IMAGE_TAG}}
PR_HELPER_BASE_IMAGE=${PR_HELPER_BASE_IMAGE:-${BASE_IMAGE_PREFIX}/pr-helper:${BASE_IMAGE_TAG}}
SIDECAR_SHIM_BASE_IMAGE=${SIDECAR_SHIM_BASE_IMAGE:-${BASE_IMAGE_PREFIX}/sidecar-shim:${BASE_IMAGE_TAG}}
TESTIMAGE_BASE_IMAGE=${TESTIMAGE_BASE_IMAGE:-${BASE_IMAGE_PREFIX}/testimage:${BASE_IMAGE_TAG}}

BUILD_ARGS="--build-arg KUBEVIRT_VERSION=${KUBEVIRT_VERSION}"
BUILD_ARGS+=" --build-arg BUILD_ARCH=${BUILD_ARCH}"
BUILD_ARGS+=" --build-arg BUILDER_IMAGE=${BUILDER_IMAGE}"
BUILD_ARGS+=" --build-arg DISTROLESS_BASE_IMAGE=${DISTROLESS_BASE_IMAGE}"

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
        example-node-hook-plugin
        test-domain-hook-sidecar
        test-helpers
        winrmcli
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
    example-node-hook-plugin)
        echo "cmd/example-node-hook-plugin/Containerfile"
        ;;
    test-domain-hook-sidecar)
        echo "cmd/plugin-sidecars/test-domain-hook/Containerfile"
        ;;
    test-helpers)
        echo "cmd/test-helpers/pod-mutator/Containerfile"
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
            DIGEST="sha256:a53fd982787799c2d8cfaa37a2b6fbac4f416437768a25d2eb246dff46bb9d79"
            ;;
        arm64)
            DIGEST="sha256:0b29f1b32b2f8d75e35de165a121a9cb211741978972f27ed47e4879c1122b18"
            ;;
        s390x)
            DIGEST="sha256:ae6d6510dfb1e1cbcf09ad85c2c0b3e58494fe10bdaa720362934422037d42a2"
            ;;
        *)
            echo "ERROR: Unsupported architecture ${normalized_arch} for fedora-with-test-tooling"
            exit 1
            ;;
        esac
        SOURCE_IMAGE="quay.io/kubevirtci/fedora-with-test-tooling@${DIGEST}"
        ;;
    alpine-with-test-tooling-container-disk)
        local normalized_arch=$(format_archname ${BUILD_ARCH} tag)
        case ${normalized_arch} in
        amd64)
            DIGEST="sha256:8c8e8bb6cd81c75e492c678abb3e5f186d52eba2174ebabc328316250acfea58"
            ;;
        arm64)
            DIGEST="sha256:5b443506b62f29f5ef5ac1bbf709338212b0b289ee2579e4feead42205685f43"
            ;;
        s390x)
            DIGEST="sha256:1a52903133c00507607e8a82308a34923e89288d852762b9f4d5da227767e965"
            ;;
        *)
            echo "ERROR: Unsupported architecture ${normalized_arch} for alpine-with-test-tooling"
            exit 1
            ;;
        esac
        SOURCE_IMAGE="quay.io/kubevirtci/alpine-with-test-tooling-container-disk@${DIGEST}"
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
            DIGEST="sha256:104545cf22d9d6ea0d675b650742f07fbffad18bdae5454e8ad79c93e0fa3eeb"
            ;;
        arm64)
            DIGEST="sha256:04b06a30b7c03994ec5c2dcd3effbefffde9f2818483dc9273035faec21e4336"
            ;;
        s390x)
            DIGEST="sha256:db633d78bf352b3109888c1d194507c2cc8940ae911fe550ee3390e4f311163a"
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
            DIGEST="sha256:1d7a44eeb45987852f796d1f4ac6a3ab9bed7ee9bfa553bb093e3b114dc72c9c"
            ;;
        arm64)
            DIGEST="sha256:c29b7231aee92c56ebaea55ca6472c41857757258d0c18037be54dd216226601"
            ;;
        s390x)
            DIGEST="sha256:007ee285eb7a789043babf323458395402dfc70c7e0b4ca3a76c2be3e42b1e2a"
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
NEEDS_SIDECAR_SHIM=false

for image in ${BUILD_TARGETS[@]}; do
    if is_retagged_image "${image}"; then
        :
    elif is_container_disk_image "${image}"; then
        CONTAINER_DISK_IMAGES+=" ${image}"
    else
        REGULAR_BUILD_IMAGES+=" ${image}"

        if [[ "${image}" == "example-hook-sidecar" || "${image}" == "example-cloudinit-hook-sidecar" || "${image}" == "example-disk-mutation-hook-sidecar" ]]; then
            NEEDS_SIDECAR_SHIM=true
        fi
    fi
done

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
        extra_args="--build-arg TESTING_BASE_IMAGE=${TESTIMAGE_BASE_IMAGE}"
    elif [[ "${image}" == "example-hook-sidecar" || "${image}" == "example-cloudinit-hook-sidecar" || "${image}" == "example-disk-mutation-hook-sidecar" ]]; then
        extra_args="--build-arg SIDECAR_SHIM_IMAGE=${DOCKER_PREFIX}/sidecar-shim:${DOCKER_TAG}"
    elif [[ "${image}" == "virt-launcher" ]]; then
        extra_args="--build-arg LAUNCHERBASE_IMAGE=${LAUNCHERBASE_IMAGE}"
        extra_args+=" --build-arg LIBVIRT_DEVEL_IMAGE=${LIBVIRT_DEVEL_IMAGE}"
    elif [[ "${image}" == "virt-handler" ]]; then
        extra_args="--build-arg HANDLERBASE_IMAGE=${HANDLERBASE_IMAGE}"
        extra_args+=" --build-arg LIBVIRT_DEVEL_IMAGE=${LIBVIRT_DEVEL_IMAGE}"
    elif [[ "${image}" == "virt-exportserver" ]]; then
        extra_args="--build-arg EXPORTSERVERBASE_IMAGE=${EXPORTSERVERBASE_IMAGE}"
    elif [[ "${image}" == "libguestfs-tools" ]]; then
        extra_args="--build-arg LIBGUESTFS_BASE_IMAGE=${LIBGUESTFS_BASE_IMAGE}"
    elif [[ "${image}" == "pr-helper" ]]; then
        extra_args="--build-arg PR_HELPER_BASE_IMAGE=${PR_HELPER_BASE_IMAGE}"
    elif [[ "${image}" == "sidecar-shim" ]]; then
        extra_args="--build-arg SIDECAR_SHIM_BASE_IMAGE=${SIDECAR_SHIM_BASE_IMAGE}"
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
