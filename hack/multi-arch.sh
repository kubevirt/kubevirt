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

COMMAND=$1

build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)

# Only add the tailing $arch when doing a multi-arch build
if [ "$build_count" -gt 1 ]; then
    for arch in ${BUILD_ARCH//,/ }; do
        echo "[INFO] -- working on $arch --"
        arch=$(format_archname $arch)
        tag=$(format_archname $arch tag)
        DOCKER_TAG=$DOCKER_TAG-$tag ARCHITECTURE=$arch hack/bazel-${COMMAND}.sh
    done
else
    arch=$(format_archname ${BUILD_ARCH})
    ARCHITECTURE=${arch} hack/bazel-${COMMAND}.sh
fi

# Push multiarch index targets that contain pre-built per-arch images.
# These are pushed once with the base tag, regardless of build architecture,
# because the OCI image index already contains all per-arch variants.
if [ "${COMMAND}" = "push-images" ]; then
    source hack/bootstrap.sh
    source hack/config.sh

    multiarch_targets="
        fedora-with-test-tooling-container-disk
    "

    host_arch=$(format_archname)
    for tag in ${docker_tag} ${docker_tag_alt}; do
        for target in ${multiarch_targets}; do
            bazel run \
                --config=${host_arch} ${BAZEL_CS_CONFIG} \
                //:push-${target} -- --repository ${docker_prefix}/${image_prefix}${target} --tag ${tag}
        done
    done

    # for the imagePrefix operator test
    if [[ $image_prefix_alt ]]; then
        for target in ${multiarch_targets}; do
            bazel run \
                --config=${host_arch} ${BAZEL_CS_CONFIG} \
                //:push-${target} -- --repository ${docker_prefix}/${image_prefix_alt}${target} --tag ${docker_tag}
        done
    fi
fi
