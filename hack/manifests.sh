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
# Copyright The KubeVirt Authors.
#

set -e

DOCKER_TAG=${DOCKER_TAG:-devel}
DOCKER_TAG_ALT=${DOCKER_TAG_ALT:-devel_alt}

echo "Building manifests..."

${KUBEVIRT_PATH}hack/dockerized "KUBEVIRT_NO_BAZEL=${KUBEVIRT_NO_BAZEL} BUILD_ARCH=${BUILD_ARCH} && CSV_VERSION=${CSV_VERSION} QUAY_REPOSITORY=${QUAY_REPOSITORY} \
	  DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} KUBEVIRT_ONLY_USE_TAGS=${KUBEVIRT_ONLY_USE_TAGS} \
	  IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} PACKAGE_NAME=${PACKAGE_NAME} KUBEVIRT_INFRA_REPLICAS=${KUBEVIRT_INFRA_REPLICAS} KUBEVIRT_E2E_PARALLEL_NODES=${KUBEVIRT_E2E_PARALLEL_NODES} \
	  KUBEVIRT_INSTALLED_NAMESPACE=${KUBEVIRT_INSTALLED_NAMESPACE} feature_gates=${FEATURE_GATES} runbook_url_template=${RUNBOOK_URL_TEMPLATE} ./hack/build-manifests.sh"

echo "Done $0"
