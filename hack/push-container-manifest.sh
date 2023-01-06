#!/usr/bin/env bash
#
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
# Copyright 2023 NVIDIA CORPORATION
#

source hack/common.sh

# No need to push manifests if using a single arch
build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)
if [ "$build_count" -lt 2 ]; then
    exit 0
fi

fail_if_cri_bin_missing

export DOCKER_CLI_EXPERIMENTAL=enabled
for image in $(find ${DIGESTS_DIR}/*/bazel-bin/ -name '*.digest' -printf '%f\n' | sed s/^push-//g | sed s/\.digest$//g | sort -u); do
    MANIFEST_IMAGES=""
    for ARCH in ${BUILD_ARCH//,/ }; do
        digest=$(cat ${DIGESTS_DIR}/${ARCH}/bazel-bin/push-$image.digest)
        MANIFEST_IMAGES="${MANIFEST_IMAGES} --amend ${DOCKER_PREFIX}/${image}@${digest}"
    done

    echo ${KUBEVIRT_CRI} manifest create ${DOCKER_PREFIX}/${image}:${DOCKER_TAG} ${MANIFEST_IMAGES}
    ${KUBEVIRT_CRI} manifest create ${DOCKER_PREFIX}/${image}:${DOCKER_TAG} ${MANIFEST_IMAGES}
    ${KUBEVIRT_CRI} manifest push ${DOCKER_PREFIX}/${image}:${DOCKER_TAG}
done
