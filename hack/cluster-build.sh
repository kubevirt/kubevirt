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
    # Container flow: build with Podman/Docker, push directly
    ${KUBEVIRT_PATH}hack/dockerized "KUBEVIRT_NO_BAZEL=true BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/go-build-functests.sh"
    BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PREFIX=${IMAGE_PREFIX} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/multi-arch-container.sh
    BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PREFIX=${IMAGE_PREFIX} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/push-images-container.sh
    BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PREFIX=${IMAGE_PREFIX} hack/push-container-manifest.sh

    # Retag and push alt images for operator upgrade tests
    _cri=$(determine_cri_bin)
    _prefix="${DOCKER_PREFIX:-${docker_prefix}}"
    _ip="${IMAGE_PREFIX:-${image_prefix}}"
    _push_flags=""
    if [[ "${_prefix}" == localhost:* ]] || [[ "${_prefix}" == registry:* ]]; then
        _push_flags="--tls-verify=false"
    fi
    _core_images="virt-operator virt-api virt-controller virt-handler virt-launcher virt-exportproxy virt-exportserver"
    if [ -n "${DOCKER_TAG_ALT}" ]; then
        for img in ${_core_images}; do
            ${_cri} tag "${_prefix}/${_ip}${img}:${DOCKER_TAG}" "${_prefix}/${_ip}${img}:${DOCKER_TAG_ALT}"
            ${_cri} push ${_push_flags} "${_prefix}/${_ip}${img}:${DOCKER_TAG_ALT}"
        done
    fi
    if [ -n "${IMAGE_PREFIX_ALT}" ]; then
        _alt="${IMAGE_PREFIX_ALT:-${image_prefix_alt}}"
        for img in ${_core_images}; do
            ${_cri} tag "${_prefix}/${_ip}${img}:${DOCKER_TAG}" "${_prefix}/${_alt}${img}:${DOCKER_TAG}"
            ${_cri} push ${_push_flags} "${_prefix}/${_alt}${img}:${DOCKER_TAG}"
        done
    fi
else
    # Bazel flow: build and push via Bazel rules
    ${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-build-functests.sh"
    ${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} ./hack/multi-arch.sh push-images"
    BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} hack/push-container-manifest.sh
fi

# Build and push cross-arch container disk images for cross-architecture emulation testing
if [ "${KUBEVIRT_CROSS_ARCH_EMULATION}" ]; then
    cross_arch_targets="fedora-with-test-tooling-container-disk alpine-with-test-tooling-container-disk"
    host_arch=$(uname -m)
    case ${host_arch} in
    x86_64) cross_arch="arm64" ;;
    aarch64) cross_arch="amd64" ;;
    esac
    if [ -n "${cross_arch}" ]; then
        cross_build_arch=$(format_archname ${cross_arch})
        cross_tag=$(format_archname ${cross_arch} tag)
        if [ "${KUBEVIRT_NO_BAZEL}" = "true" ]; then
            PUSH_TARGETS="${cross_arch_targets}" DOCKER_TAG=${DOCKER_TAG}-${cross_tag} DOCKER_TAG_ALT= DOCKER_PREFIX=${DOCKER_PREFIX} BUILD_ARCH=${cross_build_arch} IMAGE_PREFIX=${IMAGE_PREFIX} ./hack/build-images-container.sh
            PUSH_TARGETS="${cross_arch_targets}" DOCKER_TAG=${DOCKER_TAG}-${cross_tag} DOCKER_PREFIX=${DOCKER_PREFIX} BUILD_ARCH=${cross_build_arch} IMAGE_PREFIX=${IMAGE_PREFIX} ./hack/push-images-container.sh
        else
            ${KUBEVIRT_PATH}hack/dockerized "PUSH_TARGETS='${cross_arch_targets}' DOCKER_TAG=${DOCKER_TAG}-${cross_tag} DOCKER_TAG_ALT= DOCKER_PREFIX=${DOCKER_PREFIX} ARCHITECTURE=${cross_build_arch} IMAGE_PREFIX=${IMAGE_PREFIX} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-push-images.sh"
        fi
    fi
fi

# Push virt-template images
if [ "${KUBEVIRT_NO_BAZEL}" = "true" ]; then
    source hack/virt-template/default.sh
    KUBEVIRT_CRI=$(determine_cri_bin)
    vt_prefix="${DOCKER_PREFIX:-${docker_prefix}}"
    vt_image_prefix="${IMAGE_PREFIX:-${image_prefix}}"
    for target in virt-template-apiserver virt-template-controller; do
        src="${vt_prefix}/${vt_image_prefix}${target}:${DOCKER_TAG}"
        dst="${vt_prefix}/${vt_image_prefix}${target}:${virt_template_version}"
        ${KUBEVIRT_CRI} tag "${src}" "${dst}"
        ${KUBEVIRT_CRI} push "${dst}" --tls-verify=false 2>/dev/null || ${KUBEVIRT_CRI} push "${dst}"
        if [[ -n "${IMAGE_PREFIX_ALT:-${image_prefix_alt}}" ]]; then
            vt_alt="${IMAGE_PREFIX_ALT:-${image_prefix_alt}}"
            dst_alt="${vt_prefix}/${vt_alt}${target}:${virt_template_version}"
            ${KUBEVIRT_CRI} tag "${src}" "${dst_alt}"
            ${KUBEVIRT_CRI} push "${dst_alt}" --tls-verify=false 2>/dev/null || ${KUBEVIRT_CRI} push "${dst_alt}"
        fi
    done
else
    ${KUBEVIRT_PATH}hack/dockerized "BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} ./hack/virt-template/push-images.sh"
fi

echo "Done $0"
