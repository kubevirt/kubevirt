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

# Build functests (ginkgo, tests.test, junit-merger)
${KUBEVIRT_PATH}hack/dockerized "KUBEVIRT_GO_BUILD_TAGS=${KUBEVIRT_GO_BUILD_TAGS} ./hack/go-build-functests.sh"

# Build main binaries
${KUBEVIRT_PATH}hack/dockerized "KUBEVIRT_VERSION=${DOCKER_TAG} ./hack/build-go.sh install"

# Build and push images
DOCKER_PREFIX=${docker_prefix} DOCKER_TAG=${DOCKER_TAG} ${KUBEVIRT_PATH}hack/build-images.sh --push \
    virt-launcher virt-handler virt-api virt-controller virt-operator \
    virt-exportserver virt-exportproxy sidecar-shim

# Push multi-arch manifests if needed
BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} hack/push-container-manifest.sh

echo "Done $0"
