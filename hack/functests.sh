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

set -e

DOCKER_TAG=${DOCKER_TAG:-devel}
DOCKER_TAG_ALT=${DOCKER_TAG_ALT:-devel_alt}

export ARTIFACTS=${ARTIFACTS:-_out/artifacts}

source hack/common.sh
source hack/config.sh

# This git command returns the most recent tag created in a specific branch
# Example: if working in the release-0.17 branch, it will return v0.17.2 (or whatever the latest tag is today in release-0.17)
# Example: if working in master, it will return v0.18.0 (or whatever the latest tag is today)
#
# These git tags happen to correspond exactly to our container tags, so the convention works well
# for determining the release we should be testing updates with.
_default_previous_release_tag=$(git describe --tags --abbrev=0 "$(git rev-parse HEAD)")
_default_previous_release_registry="index.docker.io/kubevirt"

previous_release_tag=${PREVIOUS_RELEASE_TAG:-$_default_previous_release_tag}
previous_release_registry=${PREVIOUS_RELEASE_REGISTRY:-$_default_previous_release_registry}

functest_docker_prefix=${manifest_docker_prefix-${docker_prefix}}

if [[ ${KUBEVIRT_PROVIDER} == os-* ]] || [[ ${KUBEVIRT_PROVIDER} =~ (okd|ocp)-* ]]; then
    oc=${kubectl}
fi

${TESTS_OUT_DIR}/tests.test -kubeconfig=${kubeconfig} -container-tag=${docker_tag} -container-tag-alt=${docker_tag_alt} -container-prefix=${functest_docker_prefix} -image-prefix-alt=${image_prefix_alt} -oc-path=${oc} -kubectl-path=${kubectl} -gocli-path=${gocli} -test.timeout 420m ${FUNC_TEST_ARGS} -installed-namespace=${namespace} -previous-release-tag=${previous_release_tag} -previous-release-registry=${previous_release_registry}
