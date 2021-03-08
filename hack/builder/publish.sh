#!/usr/bin/env bash
set -ex

SCRIPT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")"
    pwd
)"

trap 'cleanup' EXIT

cleanup() {
    rm manifests/ -rf || true
}

. ${SCRIPT_DIR}/version.sh

cleanup

for ARCH in ${ARCHITECTURES}; do
    docker push quay.io/kubevirt/builder:${VERSION}-${ARCH}
    TMP_IMAGES="${TMP_IMAGES} quay.io/kubevirt/builder:${VERSION}-${ARCH}"
done

export DOCKER_CLI_EXPERIMENTAL=enabled
docker manifest create --amend quay.io/kubevirt/builder:${VERSION} ${TMP_IMAGES}
docker manifest push quay.io/kubevirt/builder:${VERSION}
