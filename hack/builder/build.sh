#!/usr/bin/env bash
set -ex

source $(dirname "$0")/../common.sh

fail_if_cri_bin_missing

SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    pwd
)"

# Detect host architecture
HOST_ARCH=$(uname -m)
case ${HOST_ARCH} in
    x86_64)
        HOST_ARCH="amd64"
        ;;
    aarch64)
        HOST_ARCH="arm64"
        ;;
    ppc64le)
        HOST_ARCH="ppc64le"
        ;;
    s390x)
        HOST_ARCH="s390x"
        ;;
esac

# If qemu-static has already been registered as a runner for foreign
# binaries, for example by installing qemu-user and qemu-user-binfmt
# packages on Fedora or by having already run this script earlier,
# then we shouldn't alter the existing configuration to avoid the
# risk of possibly breaking it.
# Note: Only setup qemu-user-static on amd64 hosts for cross-compilation.
# On native ppc64le, arm64, or s390x hosts, we don't need emulation.
if [ "${HOST_ARCH}" = "amd64" ]; then
    if ! grep -q -E '^enabled$' /proc/sys/fs/binfmt_misc/qemu-aarch64 2>/dev/null; then
        ${KUBEVIRT_CRI} >&2 run --rm --privileged docker.io/multiarch/qemu-user-static --reset -p yes
    fi
fi

# On non-amd64 hosts, only build for the native architecture unless explicitly overridden
# This is because cross-compilation requires qemu-user-static which is only available for amd64
# Set this BEFORE sourcing common.sh to prevent the default multi-arch value from being used
if [ "${HOST_ARCH}" != "amd64" ] && [ -z "${ARCHITECTURES}" ]; then
    export ARCHITECTURES="${HOST_ARCH}"
    echo >&2 "Building for native architecture only: ${HOST_ARCH}"
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
    ${KUBEVIRT_CRI} >&2 pull --platform="linux/${ARCH}" quay.io/centos/centos:stream9
    ${KUBEVIRT_CRI} >&2 build --platform="linux/${ARCH}" -t "${DOCKER_PREFIX}/${DOCKER_IMAGE}:${VERSION}-${ARCH}" --build-arg ARCH=${ARCH} --build-arg SONOBUOY_ARCH=${sonobuoy_arch} --build-arg BAZEL_ARCH=${bazel_arch} -f "${SCRIPT_DIR}/Dockerfile" "${SCRIPT_DIR}"
done

# Only build the cross-compile image if we built the amd64 builder
# The cross-compile image is used for building from amd64 to other architectures
if echo "${ARCHITECTURES}" | grep -q "amd64"; then
    ${KUBEVIRT_CRI} >&2 build --platform="linux/amd64" -t "${DOCKER_PREFIX}/${DOCKER_CROSS_IMAGE}:${VERSION}" --build-arg BUILDER_IMAGE="${DOCKER_PREFIX}/${DOCKER_IMAGE}:${VERSION}-amd64" -f "${SCRIPT_DIR}/Dockerfile.cross-compile" "${SCRIPT_DIR}"
else
    echo >&2 "Skipping cross-compile image build (amd64 builder not built)"
fi

# Print the version for use by other callers such as publish.sh
echo ${VERSION}
