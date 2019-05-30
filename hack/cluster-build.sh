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

source hack/common.sh
source cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

echo "Building ..."

# Build everyting and publish it
${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/build-manifests.sh"
${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-build.sh"
${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-push-images.sh"

echo "Done"
