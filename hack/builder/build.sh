#!/usr/bin/env bash
set -ex

SCRIPT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")"
    pwd
)"

trap 'cleanup' EXIT

cleanup() {
    docker rm -f dummy-qemu-user-static >/dev/null || true
    rm ${SCRIPT_DIR}/qemu-ppc64le-static || true
}

. ${SCRIPT_DIR}/version.sh

cleanup

docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
docker create -ti --name dummy-qemu-user-static multiarch/qemu-user-static
docker cp dummy-qemu-user-static:/usr/bin/qemu-ppc64le-static ${SCRIPT_DIR}/qemu-ppc64le-static

for ARCH in ${ARCHITECTURES}; do
    docker build -t kubevirt/builder:${VERSION}-${ARCH} --build-arg ARCH=${ARCH} -f ${SCRIPT_DIR}/Dockerfile ${SCRIPT_DIR}
    TMP_IMAGES="${TMP_IMAGES} kubevirt/builder:${VERSION}-${ARCH}"
done
