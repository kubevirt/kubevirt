#!/usr/bin/env bash
set -ex

SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    pwd
)"

trap 'cleanup' EXIT

cleanup() {
    docker rm -f dummy-qemu-user-static >/dev/null || true
    rm "${SCRIPT_DIR}/qemu-aarch64-static" || true
}

# shellcheck source=hack/builder/version.sh
. "${SCRIPT_DIR}/version.sh"

cleanup

docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
docker create -ti --name dummy-qemu-user-static multiarch/qemu-user-static
docker cp dummy-qemu-user-static:/usr/bin/qemu-aarch64-static "${SCRIPT_DIR}/qemu-aarch64-static"

for ARCH in ${ARCHITECTURES}; do
    case ${ARCH} in
    amd64)
        sonobuoy_arch="amd64"
        bazel_arch="x86_64"
        ;;
    arm64v8)
        sonobuoy_arch="arm64"
        bazel_arch="arm64"
        ;;
    *)
        sonobuoy_arch=${ARCH}
        bazel_arch=${ARCH}
        ;;
    esac
    docker pull "${ARCH}/fedora:32"
    docker build -t "quay.io/kubevirt/builder:${VERSION}-${ARCH}" --build-arg ARCH="${ARCH}" --build-arg SONOBUOY_ARCH=${sonobuoy_arch} --build-arg BAZEL_ARCH=${bazel_arch} -f "${SCRIPT_DIR}/Dockerfile" "${SCRIPT_DIR}"
    TMP_IMAGES="${TMP_IMAGES} quay.io/kubevirt/builder:${VERSION}-${ARCH}"
done
