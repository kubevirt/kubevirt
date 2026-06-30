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

set -e

source hack/common.sh
source hack/config.sh
source hack/container-utils.sh

KUBEVIRT_CRI=${KUBEVIRT_CRI:-$(determine_cri_bin)}
DOCKER_TAG=${DOCKER_TAG:-latest}
DOCKER_PREFIX=${DOCKER_PREFIX:-quay.io/kubevirt}
IMAGE_PREFIX=${IMAGE_PREFIX:-}
BUILD_OUTPUT_DIR=${BUILD_OUTPUT_DIR:-_out}
DIGESTS_DIR=${DIGESTS_DIR:-${BUILD_OUTPUT_DIR}/digests}
CACHE_DIR=${CACHE_DIR:-${BUILD_OUTPUT_DIR}/container-disks}

# Detect architecture
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

echo "==============================================="
echo "Building Container Disk Images"
echo "Architecture: ${BUILD_ARCH}"
echo "Container Engine: ${KUBEVIRT_CRI}"
echo "Tag: ${DOCKER_TAG}"
echo "Prefix: ${DOCKER_PREFIX}"
echo "Cache Directory: ${CACHE_DIR}"
echo "==============================================="

mkdir -p "${CACHE_DIR}"

build_container_disk() {
    local name=$1
    local url=$2
    local sha256=$3
    local package_dir=${4:-/disk}
    local dest_filename=${5:-$(basename "$url")}
    local filename=$(basename "$url")
    local cached_file="${CACHE_DIR}/${filename}"
    local sha_file="${CACHE_DIR}/${filename}.sha256"

    echo "Building ${name}"
    echo "URL: ${url}"
    echo "Package dir: ${package_dir}"

    # Check if cached file exists and has correct checksum
    local needs_download=true
    if [[ -f "${cached_file}" && -f "${sha_file}" ]]; then
        local cached_sha=$(cat "${sha_file}")
        if [[ "${cached_sha}" == "${sha256}" ]]; then
            echo "Verifying cached file..."
            if echo "${sha256}  ${cached_file}" | sha256sum -c - >/dev/null 2>&1; then
                echo "Using cached file (checksum verified)"
                needs_download=false
            else
                echo "Cached file corrupted, will re-download"
            fi
        else
            echo "SHA changed, will re-download"
        fi
    fi

    if [[ "${needs_download}" == "true" ]]; then
        curl -L -o "${cached_file}.tmp" "${url}"

        echo "${sha256}  ${cached_file}.tmp" | sha256sum -c - || {
            echo "ERROR: Checksum verification failed for ${name}"
            rm -f "${cached_file}.tmp"
            return 1
        }

        mv "${cached_file}.tmp" "${cached_file}"
        echo "${sha256}" >"${sha_file}"
        echo "Downloaded successfully"
    fi

    # Create Containerfile
    cat >"${CACHE_DIR}/Containerfile.${name}" <<DOCKERFILE
FROM scratch
COPY --chown=107:107 --chmod=0440 ${filename} ${package_dir}/${dest_filename}
DOCKERFILE

    # Build image
    local full_tag="${DOCKER_PREFIX}/${IMAGE_PREFIX}${name}:${DOCKER_TAG}"
    echo "Building image: ${full_tag}"
    ${KUBEVIRT_CRI} build \
        --platform linux/${BUILD_ARCH} \
        -f "${CACHE_DIR}/Containerfile.${name}" \
        -t "${full_tag}" \
        "${CACHE_DIR}/"

    save_image_digest "${name}" "${full_tag}" "${BUILD_ARCH}"

    rm -f "${CACHE_DIR}/Containerfile.${name}"

    echo "Successfully built ${full_tag}"
}

case ${BUILD_ARCH} in
amd64)
    build_container_disk "alpine-container-disk-demo" \
        "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/x86_64/alpine-virt-3.20.1-x86_64.iso" \
        "f87a0fd3ab0e65d2a84acd5dad5f8b6afce51cb465f65dd6f8a3810a3723b6e4"

    build_container_disk "cirros-container-disk-demo" \
        "https://download.cirros-cloud.net/0.5.2/cirros-0.5.2-x86_64-disk.img" \
        "932fcae93574e242dc3d772d5235061747dfe537668443a1f0567d893614b464"

    build_container_disk "cirros-custom-container-disk-demo" \
        "https://download.cirros-cloud.net/0.5.2/cirros-0.5.2-x86_64-disk.img" \
        "932fcae93574e242dc3d772d5235061747dfe537668443a1f0567d893614b464" \
        "/custom-disk" \
        "downloaded"
    ;;
arm64)
    build_container_disk "alpine-container-disk-demo" \
        "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/aarch64/alpine-virt-3.20.1-aarch64.iso" \
        "ca2f0e8aa7a1d7917bce7b9e7bd413772b64ec529a1938d20352558f90a5035a"

    build_container_disk "cirros-container-disk-demo" \
        "https://download.cirros-cloud.net/0.5.2/cirros-0.5.2-aarch64-disk.img" \
        "889c1117647b3b16cfc47957931c6573bf8e755fc9098fdcad13727b6c9f2629"

    build_container_disk "cirros-custom-container-disk-demo" \
        "https://download.cirros-cloud.net/0.5.2/cirros-0.5.2-aarch64-disk.img" \
        "889c1117647b3b16cfc47957931c6573bf8e755fc9098fdcad13727b6c9f2629" \
        "/custom-disk" \
        "downloaded"
    ;;
s390x)
    build_container_disk "alpine-container-disk-demo" \
        "https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/s390x/alpine-standard-3.18.8-s390x.iso" \
        "4ca1462252246d53e4949523b87fcea088e8b4992dbd6df792818c5875069b16"

    # Cirros not available for s390x
    echo "Skipping cirros images (not available for s390x)"
    ;;
esac

if [[ "${BUILD_ARCH}" != "s390x" ]]; then
    build_container_disk "virtio-container-disk" \
        "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.266-1/virtio-win-0.1.266.iso" \
        "57b0f6dc8dc92dc2ae8621f8b1bfbd8a873de9bedc788c4c4b305ea28acc77cd"
fi

echo "Container disk builds complete"
