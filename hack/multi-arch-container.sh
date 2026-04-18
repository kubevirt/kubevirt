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
# Copyright 2026 The KubeVirt Authors.
#

set -e

source hack/common.sh
source hack/container-utils.sh

build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)

echo "==============================================="
echo "Multi-arch Container Build"
echo "==============================================="
echo "Architectures: ${BUILD_ARCH}"
echo "Build count: ${build_count}"
echo "Container engine: ${KUBEVIRT_CRI:-auto-detect}"
echo "==============================================="

# Build for each architecture
if [ "$build_count" -gt 1 ]; then
    for arch in ${BUILD_ARCH//,/ }; do
        echo ""
        echo "Building for architecture: $arch"

        arch=$(format_archname $arch)
        tag=$(format_archname $arch tag)

        BUILD_ARCH=$arch \
            DOCKER_TAG=$DOCKER_TAG-$tag \
            DOCKER_PREFIX=${DOCKER_PREFIX} \
            IMAGE_PREFIX=${IMAGE_PREFIX} \
            KUBEVIRT_CRI=${KUBEVIRT_CRI} \
            BUILDER_IMAGE=${BUILDER_IMAGE} \
            ./hack/build-images-container.sh
    done

    echo "Multi-arch container build completed successfully"
else
    echo "Single architecture build: ${BUILD_ARCH}"

    arch=$(format_archname ${BUILD_ARCH})

    BUILD_ARCH=${arch} \
        DOCKER_TAG=${DOCKER_TAG} \
        DOCKER_PREFIX=${DOCKER_PREFIX} \
        IMAGE_PREFIX=${IMAGE_PREFIX} \
        KUBEVIRT_CRI=${KUBEVIRT_CRI} \
        BUILDER_IMAGE=${BUILDER_IMAGE} \
        ./hack/build-images-container.sh

    echo ""
    echo "Single-arch container build completed successfully"
fi
