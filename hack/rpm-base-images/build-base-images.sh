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
# Build RPM base images for KubeVirt components.
# These images contain only the RPM-based filesystem layers and are
# rebuilt only when RPM dependencies change.
#
# Uses standalone bazeldnf to generate RPM tars (no Bazel required).

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source "${REPO_ROOT}/hack/common.sh"
source "${SCRIPT_DIR}/generate-rpm-tars.sh"

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

DOCKER_PREFIX=${DOCKER_PREFIX:-quay.io/kubevirt}
BASE_IMAGE_TAG=${BASE_IMAGE_TAG:-bazeldnf}
BUILDER_VERSION=$(grep '^kubevirt_builder_version=' "${REPO_ROOT}/hack/dockerized" | cut -d'"' -f2)
BUILDER_IMAGE=${BUILDER_IMAGE:-quay.io/kubevirt/builder:${BUILDER_VERSION}}
CENTOS_STREAM_VERSION=${CENTOS_STREAM_VERSION:-9}

build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)

echo "==============================================="
echo "Building RPM Base Images (no Bazel)"
echo "==============================================="
echo "Architecture(s): ${BUILD_ARCH}"
echo "Multi-arch: $([ "$build_count" -gt 1 ] && echo "YES" || echo "NO")"
echo "Container Engine: ${KUBEVIRT_CRI}"
echo "Registry: ${DOCKER_PREFIX}"
echo "Tag: ${BASE_IMAGE_TAG}"
echo "Builder: ${BUILDER_IMAGE}"
echo "CentOS Stream: ${CENTOS_STREAM_VERSION}"
echo "==============================================="

# Map BUILD_ARCH to the Bazel arch name used in rpmtree rules
arch_to_bazel_arch() {
    local arch=$1
    local arch_tag=$(format_archname ${arch} tag)
    case ${arch_tag} in
        amd64) echo "x86_64" ;;
        arm64) echo "aarch64" ;;
        s390x) echo "s390x" ;;
        *) echo "ERROR: Unknown arch ${arch_tag}" >&2; return 1 ;;
    esac
}

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

# Get the list of rpmtree targets that need tars generated for a given image and arch
get_rpmtree_targets_for_image() {
    local image=$1
    local bazel_arch=$2
    local cs_version=$3

    local targets="${image}_${bazel_arch}_cs${cs_version}"

    # handlerbase also needs passt_tree for the passt-repair binary
    if [[ "${image}" == "handlerbase" ]]; then
        targets="${targets} passt_tree_${bazel_arch}_cs${cs_version}"
    fi

    echo "${targets}"
}

# Generate all RPM tars needed for a given architecture
generate_tars_for_arch() {
    local arch=$1
    local bazel_arch
    bazel_arch=$(arch_to_bazel_arch ${arch})

    local -a arch_images
    read -ra arch_images <<<"$(get_base_images_for_arch ${arch})"

    echo ""
    echo "=========================================="
    echo "Generating RPM tars for ${bazel_arch}"
    echo "=========================================="

    local generated_targets=()
    for image in "${arch_images[@]}"; do
        local targets
        targets=$(get_rpmtree_targets_for_image "${image}" "${bazel_arch}" "${CENTOS_STREAM_VERSION}")

        for target in ${targets}; do
            local already_done=false
            for done_target in "${generated_targets[@]}"; do
                if [[ "${done_target}" == "${target}" ]]; then
                    already_done=true
                    break
                fi
            done
            if [[ "${already_done}" == true ]]; then
                continue
            fi

            generate_rpm_tar "${target}"
            generated_targets+=("${target}")
        done

        # libguestfs-tools also needs the appliance layer (not an rpmtree)
        if [[ "${image}" == "libguestfs-tools" ]]; then
            generate_appliance_tar "${bazel_arch}" || echo "WARNING: appliance layer skipped"
        fi
    done
}

# Build container images for a given architecture
build_for_arch() {
    local arch=$1
    local arch_normalized=$(format_archname ${arch})
    local arch_tag=$(format_archname ${arch} tag)

    local -a arch_images
    read -ra arch_images <<<"$(get_base_images_for_arch ${arch})"

    echo ""
    echo "=========================================="
    echo "Building container images for ${arch_tag}"
    echo "=========================================="
    echo "Targets: ${arch_images[*]}"

    local arch_build_args="--build-arg BUILDER_IMAGE=${BUILDER_IMAGE}"
    arch_build_args+=" --build-arg BUILD_ARCH=${arch_normalized}"
    arch_build_args+=" --build-arg CENTOS_STREAM_VERSION=${CENTOS_STREAM_VERSION}"

    for image in "${arch_images[@]}"; do
        containerfile="${SCRIPT_DIR}/Containerfile.${image}"

        if [[ ! -f "${containerfile}" ]]; then
            echo "ERROR: Containerfile not found: ${containerfile}"
            exit 1
        fi

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

# Main flow: generate tars first, then build images
for arch in ${BUILD_ARCH//,/ }; do
    generate_tars_for_arch "${arch}"
done

for arch in ${BUILD_ARCH//,/ }; do
    build_for_arch "${arch}"
done

echo ""
echo "==============================================="
echo "Base image build completed successfully!"
echo "==============================================="
