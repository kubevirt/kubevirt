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

# This logic moved into the Makefile.
# We're leaving this file around for people who still reference this
# specific script in their development workflow.

set -e

DOCKER_TAG=${DOCKER_TAG:-devel}
DOCKER_TAG_ALT=${DOCKER_TAG_ALT:-devel_alt}

source hack/common.sh
source cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

echo "Building ..."

# Build everyting and publish it
${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/build-func-tests.sh"
${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} ./hack/bazel-push-images.sh"
${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} IMAGE_PREFIX=${IMAGE_PREFIX}  IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} feature_gates=${FEATURE_GATES} KUBEVIRT_ONLY_USE_TAGS=${KUBEVIRT_ONLY_USE_TAGS} ./hack/build-manifests.sh"

echo "Done $0"
