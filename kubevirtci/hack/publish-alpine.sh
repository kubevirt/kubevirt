#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/detect_cri.sh"

export KUBEVIRTCI_TAG=${KUBEVIRTCI_TAG:-$(date +"%y%m%d%H%M")-$(git rev-parse --short HEAD)}
export CRI_BIN=${CRI_BIN:-$(detect_cri)}

if [ -z "${CRI_BIN}" ]; then
    echo "ERROR: Neither podman nor docker is available." >&2
    exit 1
fi

# s390x requires native hardware because alpine-make-vm-image runs zipl (the
# s390x bootloader installer) in a chroot. zipl performs low-level ioctl calls
# on the block device to write IPL boot records, which segfaults under
# qemu-user-static. arm64 cross-builds fine because its UEFI bootloader setup
# is just writing a text file (startup.nsh), with no low-level disk operations.
if [ "$(uname -m)" = "s390x" ]; then
    BUILD_ARCHES=${BUILD_ARCHES:-"s390x"}
else
    BUILD_ARCHES=${BUILD_ARCHES:-"amd64 arm64"}
fi
TARGET_REPO=${TARGET_REPO:-"quay.io/kubevirtci"}
TARGET_KUBEVIRT_REPO=${TARGET_KUBEVIRT_REPO:-"quay.io/kubevirt"}
IMAGE_NAME="alpine-with-test-tooling-container-disk"

function build_and_push_alpine() {
    local arch=$1
    local tagged_image="${TARGET_REPO}/${IMAGE_NAME}:${KUBEVIRTCI_TAG}-${arch}"
    local devel_image="${TARGET_KUBEVIRT_REPO}/${IMAGE_NAME}:devel-${arch}"

    echo "INFO: building alpine container disk for ${arch}"
    (cd cluster-provision/images/vm-image-builder && ARCHITECTURE=${arch} ./create-containerdisk.sh alpine-cloud-init)

    ${CRI_BIN} tag alpine-cloud-init:devel "${tagged_image}"
    ${CRI_BIN} tag alpine-cloud-init:devel "${devel_image}"

    echo "INFO: pushing ${tagged_image}"
    ${CRI_BIN} push "${tagged_image}"
    echo "INFO: pushing ${devel_image}"
    ${CRI_BIN} push "${devel_image}"
}

function create_and_push_manifest() {
    local base_name=$1
    local repo=$2

    local manifest_name="${repo}/${IMAGE_NAME}:${base_name}"
    local images=""

    for arch in ${BUILD_ARCHES}; do
        images="${images} ${repo}/${IMAGE_NAME}:${base_name}-${arch}"
    done

    if [ -z "${images}" ]; then
        echo "WARN: no images to include in manifest ${manifest_name}, skipping"
        return
    fi

    if ${CRI_BIN} manifest exists "${manifest_name}" 2>/dev/null; then
        ${CRI_BIN} manifest rm "${manifest_name}"
    fi

    echo "INFO: creating manifest ${manifest_name}"
    ${CRI_BIN} manifest create "${manifest_name}" ${images}
    echo "INFO: pushing manifest ${manifest_name}"
    ${CRI_BIN} manifest push "${manifest_name}"
}

for arch in ${BUILD_ARCHES}; do
    build_and_push_alpine "${arch}"
done

create_and_push_manifest "${KUBEVIRTCI_TAG}" "${TARGET_REPO}"
create_and_push_manifest "devel" "${TARGET_KUBEVIRT_REPO}"
