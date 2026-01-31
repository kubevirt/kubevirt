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

# Source virt-template version for its images
source hack/virt-template/default.sh

# No need to push manifests if using a single arch
build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)
if [ "$build_count" -lt 2 ]; then
    exit 0
fi

fail_if_cri_bin_missing

# Get tag for an image (virt-template uses its own version, others use DOCKER_TAG)
function get_tag_for_image() {
    local image=$1
    if is_virt_template_target "${image}"; then
        echo "${virt_template_version}"
    else
        echo "${DOCKER_TAG}"
    fi
}

function podman_push_manifest() {
    local image=$1
    local tag=$(get_tag_for_image "${image}")
    # FIXME: Workaround https://github.com/containers/podman/issues/18360 and remove once https://github.com/containers/podman/commit/bab4217cd16be609ac35ccf3061d1e34f787856f is released
    echo ${KUBEVIRT_CRI} manifest create ${DOCKER_PREFIX}/${image}:${tag}
    ${KUBEVIRT_CRI} manifest create ${DOCKER_PREFIX}/${image}:${tag}
    for ARCH in ${BUILD_ARCH//,/ }; do
        FORMATTED_ARCH=$(format_archname ${ARCH} tag)
        TAGGED_IMAGE="${DOCKER_PREFIX}/${image}:${tag}-${FORMATTED_ARCH}"
        if skopeo inspect docker://${TAGGED_IMAGE} &>/dev/null; then
            ${KUBEVIRT_CRI} manifest add ${DOCKER_PREFIX}/${image}:${tag} ${TAGGED_IMAGE}
        else
            echo "Warning: Image ${TAGGED_IMAGE} does not exist, skipping"
        fi
    done
    ${KUBEVIRT_CRI} manifest push --all ${DOCKER_PREFIX}/${image}:${tag} ${DOCKER_PREFIX}/${image}:${tag}
}

function docker_push_manifest() {
    local image=$1
    local tag=$(get_tag_for_image "${image}")
    MANIFEST_IMAGES=""
    for ARCH in ${BUILD_ARCH//,/ }; do
        FORMATTED_ARCH=$(format_archname ${ARCH} tag)
        TAGGED_IMAGE="${DOCKER_PREFIX}/${image}:${tag}-${FORMATTED_ARCH}"
        if skopeo inspect docker://${TAGGED_IMAGE} &>/dev/null; then
            MANIFEST_IMAGES="${MANIFEST_IMAGES} --amend ${TAGGED_IMAGE}"
        else
            echo "Warning: Image ${TAGGED_IMAGE} does not exist, skipping"
        fi
    done
    echo ${KUBEVIRT_CRI} manifest create ${DOCKER_PREFIX}/${image}:${tag} ${MANIFEST_IMAGES}
    ${KUBEVIRT_CRI} manifest create ${DOCKER_PREFIX}/${image}:${tag} ${MANIFEST_IMAGES}
    ${KUBEVIRT_CRI} manifest push ${DOCKER_PREFIX}/${image}:${tag}
}

export DOCKER_CLI_EXPERIMENTAL=enabled
for image in $(find ${DIGESTS_DIR} -name '*.image' -printf '%f\n' | sed s/\.image$//g | sort -u); do
    if [ "${KUBEVIRT_CRI}" = "podman" ]; then
        podman_push_manifest $image
    else
        docker_push_manifest $image
    fi
done
