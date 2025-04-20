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
    cd "$(dirname "$BASH_SOURCE[0]")"
    pwd
)"

trap 'cleanup' EXIT

cleanup() {
    rm manifests/ -rf || true
}

cleanup

# shellcheck source=hack/builder/common.sh
. "${SCRIPT_DIR}/common.sh"
# Use the value of VERSION returned from build.sh instead of version.sh to ensure the correct image is published
VERSION=$($(dirname "$0")/build.sh)

for ARCH in ${ARCHITECTURES}; do
    ${KUBEVIRT_CRI} push ${DOCKER_PREFIX}/${DOCKER_IMAGE}:${VERSION}-${ARCH}
    TMP_IMAGES="${TMP_IMAGES} ${DOCKER_PREFIX}/${DOCKER_IMAGE}:${VERSION}-${ARCH}"
done

export DOCKER_CLI_EXPERIMENTAL=enabled
${KUBEVIRT_CRI} manifest create --amend ${DOCKER_PREFIX}/${DOCKER_IMAGE}:${VERSION} ${TMP_IMAGES}
${KUBEVIRT_CRI} manifest push ${DOCKER_PREFIX}/${DOCKER_IMAGE}:${VERSION}

${KUBEVIRT_CRI} push ${DOCKER_PREFIX}/${DOCKER_CROSS_IMAGE}:${VERSION}
