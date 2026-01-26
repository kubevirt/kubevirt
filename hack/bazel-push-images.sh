#!/bin/bash
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
# Copyright 2019 Red Hat, Inc.
#

set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

# Source virt-template version for its images
source hack/virt-template/default.sh

# Build core images for all architectures
default_targets="
    virt-operator
    virt-api
    virt-controller
    virt-handler
    virt-launcher
    virt-exportserver
    virt-exportproxy
    virt-synchronization-controller
    alpine-container-disk-demo
    fedora-with-test-tooling-container-disk
    vm-killer
    sidecar-shim
    disks-images-provider
    libguestfs-tools
    virt-template-apiserver
    virt-template-controller
"

# Add additional images for non-s390x architectures only
if [[ "${ARCHITECTURE}" != "s390x" && "${ARCHITECTURE}" != "crossbuild-s390x" ]]; then
    default_targets+="
        conformance
        pr-helper
        example-hook-sidecar
        example-disk-mutation-hook-sidecar
        example-cloudinit-hook-sidecar
        cirros-container-disk-demo
        cirros-custom-container-disk-demo
        virtio-container-disk
        alpine-ext-kernel-boot-demo
        alpine-with-test-tooling-container-disk
        fedora-realtime-container-disk
        winrmcli
        network-slirp-binding
        network-passt-binding
        network-passt-binding-cni
    "
fi

PUSH_TARGETS=(${PUSH_TARGETS:-${default_targets}})

# Get tags to push for a target (virt-template uses its own version, others use docker_tag/docker_tag_alt)
function get_tags_for_target() {
    local target=$1
    if is_virt_template_target "${target}"; then
        echo "${virt_template_version}"
    else
        echo "${docker_tag} ${docker_tag_alt}"
    fi
}

for target in ${PUSH_TARGETS[@]}; do
    for tag in $(get_tags_for_target "${target}"); do
        bazel run \
            --config=${ARCHITECTURE} \
            //:push-${target} -- --repository ${docker_prefix}/${image_prefix}${target} --tag ${tag}

    done
done

# for the imagePrefix operator test
if [[ $image_prefix_alt ]]; then
    for target in ${PUSH_TARGETS[@]}; do

        if is_virt_template_target "${target}"; then
            tag=${virt_template_version}
        else
            tag=${docker_tag}
        fi
        bazel run \
            --config=${ARCHITECTURE} \
            //:push-${target} -- --repository ${docker_prefix}/${image_prefix_alt}${target} --tag ${tag}

    done
fi

rm -rf ${DIGESTS_DIR}/${ARCHITECTURE}
mkdir -p ${DIGESTS_DIR}/${ARCHITECTURE}

for target in ${PUSH_TARGETS[@]}; do
    dir=${DIGESTS_DIR}/${ARCHITECTURE}/${target}
    mkdir -p ${dir}
    touch ${dir}/${target}.image
done
