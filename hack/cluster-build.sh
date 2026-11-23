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
DOCKER_TAG_ALT=${DOCKER_TAG_ALT-devel_alt}
BUILD_ARCH=${BUILD_ARCH:-$(uname -m)}

source hack/common.sh
source kubevirtci/cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

echo "Building ..."

# Build the test binary outputs used by the deployed test infrastructure.
${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-build-functests.sh"
push_target_env=""
[[ -n ${PUSH_TARGETS:-} ]] && push_target_env+=" PUSH_TARGETS='${PUSH_TARGETS}'"
[[ -v PUSH_TARGETS_ALT ]] && push_target_env+=" PUSH_TARGETS_ALT='${PUSH_TARGETS_ALT}'"
registry_env=""
if [[ "${CLUSTER_SYNC_NO_PUSH:-}" == "true" ]]; then
    if [[ -z "${DOCKER_PREFIX:-}" || -z "${BAZEL_REGISTRY_ADVERTISE:-}" ]]; then
        echo "Set DOCKER_PREFIX and BAZEL_REGISTRY_ADVERTISE to a cluster-reachable Bazel registry address" >&2
        exit 2
    fi
    if [[ "${KUBEVIRT_CROSS_ARCH_EMULATION:-}" == "true" ]]; then
        echo "CLUSTER_SYNC_NO_PUSH does not yet support cross-architecture emulation images" >&2
        exit 2
    fi
    registry_env=" BAZEL_IMAGE_REGISTRY=1 BAZEL_REGISTRY_LISTEN_ADDRESS='${BAZEL_REGISTRY_LISTEN_ADDRESS:-0.0.0.0}' BAZEL_REGISTRY_PORT='${BAZEL_REGISTRY_PORT:-5001}' BAZEL_REGISTRY_ADVERTISE='${BAZEL_REGISTRY_ADVERTISE:-127.0.0.1}' BAZEL_REGISTRY_REPOSITORY_PREFIX='${BAZEL_REGISTRY_REPOSITORY_PREFIX:-kubevirt/${IMAGE_PREFIX:-}}' VIRT_TEMPLATE_VERSION='${VIRT_TEMPLATE_VERSION:-v0.3.0}'"
fi
${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT}${push_target_env}${registry_env} ./hack/multi-arch.sh push-images"
if [[ "${CLUSTER_SYNC_NO_PUSH:-}" != "true" ]]; then
    BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} hack/push-container-manifest.sh
fi

# Build and push cross-arch container disk images for cross-architecture emulation testing
if [ "${KUBEVIRT_CROSS_ARCH_EMULATION}" ] && [[ "${CLUSTER_SYNC_NO_PUSH:-}" != "true" ]]; then
    cross_arch_targets="fedora-with-test-tooling-container-disk alpine-with-test-tooling-container-disk"
    host_arch=$(uname -m)
    case ${host_arch} in
    x86_64) cross_arch="arm64" ;;
    aarch64) cross_arch="amd64" ;;
    esac
    if [ -n "${cross_arch}" ]; then
        cross_build_arch=$(format_archname ${cross_arch})
        cross_tag=$(format_archname ${cross_arch} tag)
        ${KUBEVIRT_PATH}hack/dockerized "PUSH_TARGETS='${cross_arch_targets}' DOCKER_TAG=${DOCKER_TAG}-${cross_tag} DOCKER_TAG_ALT= DOCKER_PREFIX=${DOCKER_PREFIX} ARCHITECTURE=${cross_build_arch} IMAGE_PREFIX=${IMAGE_PREFIX} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-push-images.sh"
    fi
fi

# Push virt-template images, or include their Bazel layouts in the read-only registry.
if [[ "${CLUSTER_SYNC_NO_PUSH:-}" != "true" ]]; then
    ${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} ./hack/virt-template/push-images.sh"
fi

echo "Done $0"
