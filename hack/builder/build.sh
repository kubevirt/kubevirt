#!/usr/bin/env bash
set -ex

source $(dirname "$0")/../common.sh

fail_if_cri_bin_missing

SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    pwd
)"

# shellcheck source=hack/builder/version.sh
. "${SCRIPT_DIR}/version.sh"

${KUBEVIRT_CRI} run --rm --privileged multiarch/qemu-user-static --reset -p yes

for ARCH in ${ARCHITECTURES}; do
    case ${ARCH} in
    amd64)
        sonobuoy_arch="amd64"
        bazel_arch="x86_64"
        ;;
    arm64)
        sonobuoy_arch="arm64"
        bazel_arch="arm64"
        ;;
    *)
        sonobuoy_arch=${ARCH}
        bazel_arch=${ARCH}
        ;;
    esac
    ${KUBEVIRT_CRI} pull --platform="linux/${ARCH}" quay.io/centos/centos:stream8
    ${KUBEVIRT_CRI} build --platform="linux/${ARCH}" -t "quay.io/kubevirt/builder:${VERSION}-${ARCH}" --build-arg SONOBUOY_ARCH=${sonobuoy_arch} --build-arg BAZEL_ARCH=${bazel_arch} -f "${SCRIPT_DIR}/Dockerfile" "${SCRIPT_DIR}"
    TMP_IMAGES="${TMP_IMAGES} quay.io/kubevirt/builder:${VERSION}-${ARCH}"
done
