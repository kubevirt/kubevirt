#!/usr/bin/env bash
set -ex

source $(dirname "$0")/../common.sh

fail_if_cri_bin_missing

SCRIPT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")"
    pwd
)"

trap 'cleanup' EXIT

cleanup() {
    rm manifests/ -rf || true
}

cleanup

# shellcheck source=hack/builder/arch.sh
. "${SCRIPT_DIR}/arch.sh"
# Use the value of VERSION returned from build.sh instead of version.sh to ensure the correct image is published
VERSION=$($(dirname "$0")/build.sh)

for ARCH in ${ARCHITECTURES}; do
    ${KUBEVIRT_CRI} push quay.io/kubevirt/builder:${VERSION}-${ARCH}
    TMP_IMAGES="${TMP_IMAGES} quay.io/kubevirt/builder:${VERSION}-${ARCH}"
done

export DOCKER_CLI_EXPERIMENTAL=enabled
${KUBEVIRT_CRI} manifest create --amend quay.io/kubevirt/builder:${VERSION} ${TMP_IMAGES}
${KUBEVIRT_CRI} manifest push quay.io/kubevirt/builder:${VERSION}
