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

# If qemu-static has already been registered as a runner for foreign
# binaries, for example by installing qemu-user and qemu-user-binfmt
# packages on Fedora or by having already run this script earlier,
# then we shouldn't alter the existing configuration to avoid the
# risk of possibly breaking it
if ! grep -E '^enabled$' /proc/sys/fs/binfmt_misc/qemu-aarch64 2>/dev/null; then
    ${KUBEVIRT_CRI} run --rm --privileged multiarch/qemu-user-static --reset -p yes
fi

for ARCH in ${ARCHITECTURES}; do
    case ${ARCH} in
    amd64)
        sonobuoy_arch="amd64"
        bazel_arch="x86_64"
        ;;
    *)
        sonobuoy_arch=${ARCH}
        bazel_arch=${ARCH}
        ;;
    esac
    ${KUBEVIRT_CRI} pull --platform="linux/${ARCH}" quay.io/centos/centos:stream9
    ${KUBEVIRT_CRI} build --platform="linux/${ARCH}" -t "quay.io/kubevirt/builder:${VERSION}-${ARCH}" --build-arg SONOBUOY_ARCH=${sonobuoy_arch} --build-arg BAZEL_ARCH=${bazel_arch} -f "${SCRIPT_DIR}/Dockerfile" "${SCRIPT_DIR}"
done
