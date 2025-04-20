#!/usr/bin/env bash
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
# Copyright The KubeVirt Authors.
#

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
# risk of possibly breaking it
if ! grep -q -E '^enabled$' /proc/sys/fs/binfmt_misc/qemu-aarch64 2>/dev/null; then
    ${KUBEVIRT_CRI} >&2 run --rm --privileged docker.io/multiarch/qemu-user-static --reset -p yes
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

${KUBEVIRT_CRI} >&2 build --platform="linux/amd64" -t "${DOCKER_PREFIX}/${DOCKER_CROSS_IMAGE}:${VERSION}" --build-arg BUILDER_IMAGE="${DOCKER_PREFIX}/${DOCKER_IMAGE}:${VERSION}-amd64" -f "${SCRIPT_DIR}/Dockerfile.cross-compile" "${SCRIPT_DIR}"

# Print the version for use by other callers such as publish.sh
echo ${VERSION}
