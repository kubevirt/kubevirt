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
# Copyright 2017 Red Hat, Inc.
#

# This script is called by cluster-sync.sh

set -e

DOCKER_TAG=${DOCKER_TAG:-devel}
DOCKER_TAG_ALT=${DOCKER_TAG_ALT:-devel_alt}

source hack/common.sh
source kubevirtci/cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

echo "Building ..."

if [ "${KUBEVIRT_NO_BAZEL}" = "true" ]; then
    echo "==============================================="
    echo "Using Container Build (Podman/Docker)"
    echo "KUBEVIRT_NO_BAZEL=true"
    echo "==============================================="

    # Set common environment variables for container build
    if [ -z "${DOCKER_PREFIX}" ]; then
        if [ -n "${docker_prefix}" ]; then
            DOCKER_PREFIX="${docker_prefix}"
            echo "Using registry from provider config: ${DOCKER_PREFIX}"
        else
            DOCKER_PREFIX="quay.io/kubevirt"
            echo "Using default registry: ${DOCKER_PREFIX}"
        fi
    else
        echo "Using DOCKER_PREFIX from environment: ${DOCKER_PREFIX}"
    fi

    export BUILD_ARCH=${BUILD_ARCH}
    export DOCKER_PREFIX
    export DOCKER_TAG
    export IMAGE_PREFIX
    export KUBEVIRT_CRI=${KUBEVIRT_CRI:-$(determine_cri_bin)}
    export BUILDER_IMAGE

    echo ""
    echo "Building functional test binaries"
    ${KUBEVIRT_PATH}hack/dockerized "export KUBEVIRT_NO_BAZEL=true && KUBEVIRT_GO_BUILD_TAGS=${KUBEVIRT_GO_BUILD_TAGS} ./hack/go-build-functests.sh"

    echo "Building container images"
    ${KUBEVIRT_PATH}hack/multi-arch-container.sh

    echo ""
    echo "Pushing images to cluster registry"
    ${KUBEVIRT_PATH}hack/push-images-container.sh

    if [ -n "${DOCKER_TAG_ALT}" ]; then
        echo ""
        echo "Pushing images with alt tag"

        # First re-tag images with alt tag
        for image in virt-operator virt-api virt-controller virt-handler virt-launcher virt-exportserver virt-exportproxy; do
            ${KUBEVIRT_CRI} tag \
                ${DOCKER_PREFIX}/${IMAGE_PREFIX}${image}:${DOCKER_TAG} \
                ${DOCKER_PREFIX}/${IMAGE_PREFIX_ALT}${image}:${DOCKER_TAG_ALT}
        done

        # Push with alt tag and prefix
        DOCKER_TAG=${DOCKER_TAG_ALT} \
            IMAGE_PREFIX=${IMAGE_PREFIX_ALT} \
            PUSH_TARGETS="virt-operator virt-api virt-controller virt-handler virt-launcher virt-exportserver virt-exportproxy" \
            ${KUBEVIRT_PATH}hack/push-images-container.sh
    fi

    echo ""
    echo "Creating multi-arch manifests"
    hack/push-container-manifest.sh

else
    echo "==============================================="
    echo "Using Bazel Build"
    echo "==============================================="

    # Build everything and publish it (existing Bazel workflow)
    ${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-build-functests.sh"
    ${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} ./hack/multi-arch.sh push-images"
    BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} hack/push-container-manifest.sh
fi

# Push virt-template images (Bazel path only - container build handles these in push-images-container.sh)
if [ "${KUBEVIRT_NO_BAZEL}" != "true" ]; then
    ${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} ./hack/virt-template/push-images.sh"
fi

echo "Done $0"
