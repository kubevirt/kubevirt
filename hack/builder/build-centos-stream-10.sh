#!/usr/bin/env bash
# Build CentOS Stream 10 based builder images
set -ex

source $(dirname "$0")/../common.sh

fail_if_cri_bin_missing

SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    pwd
)"

# If qemu-static has already been registered as a runner for foreign
# binaries, for example by installing qemu-user and qemu-user-binfmt
# packages on Fedora or by having already run this script earlier,
# then we shouldn't alter the existing configuration to avoid the
# risk of possibly breaking it.
# Note: Only setup qemu-user-static on amd64 hosts for cross-compilation.
# On native ppc64le, arm64, or s390x hosts, we don't need emulation.
HOST_ARCH=$(uname -m)
if [ "${HOST_ARCH}" = "x86_64" ]; then
    if ! grep -q -E '^enabled$' /proc/sys/fs/binfmt_misc/qemu-aarch64 2>/dev/null; then
        ${KUBEVIRT_CRI} >&2 run --rm --privileged quay.io/linuxserver.io/qemu-static --reset -p yes
    fi
fi

# On non-amd64 hosts, only build for the native architecture unless explicitly overridden
if [ "${HOST_ARCH}" = "ppc64le" ] || [ "${HOST_ARCH}" = "aarch64" ] || [ "${HOST_ARCH}" = "s390x" ]; then
    NATIVE_ARCH="${HOST_ARCH}"
    if [ -z "${ARCHITECTURES}" ]; then
        export ARCHITECTURES="${NATIVE_ARCH}"
        echo >&2 "Building for native architecture only: ${NATIVE_ARCH}"
    fi
fi

# shellcheck source=hack/builder/common.sh
. "${SCRIPT_DIR}/common.sh"
# shellcheck source=hack/builder/version.sh
. "${SCRIPT_DIR}/version.sh"

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
    ${KUBEVIRT_CRI} >&2 pull --platform="linux/${ARCH}" quay.io/centos/centos:stream10
    ${KUBEVIRT_CRI} >&2 build --platform="linux/${ARCH}" -t "${DOCKER_PREFIX}/${DOCKER_CS10_IMAGE}:${VERSION}-${ARCH}" --build-arg ARCH=${ARCH} --build-arg SONOBUOY_ARCH=${sonobuoy_arch} --build-arg BAZEL_ARCH=${bazel_arch} -f "${SCRIPT_DIR}/Dockerfile.cs10" "${SCRIPT_DIR}"
done

${KUBEVIRT_CRI} >&2 build --platform="linux/amd64" -t "${DOCKER_PREFIX}/${DOCKER_CS10_CROSS_IMAGE}:${VERSION}" --build-arg BUILDER_IMAGE="${DOCKER_PREFIX}/${DOCKER_CS10_IMAGE}:${VERSION}-amd64" -f "${SCRIPT_DIR}/Dockerfile.cross-compile" "${SCRIPT_DIR}"

# Print the version for use by other callers such as publish.sh
echo ${VERSION}
