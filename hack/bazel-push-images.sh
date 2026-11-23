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

# Core images for all architectures
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
    test-helpers
    alpine-with-test-tooling-container-disk
"

# Add additional images for s390x only
if [[ "${ARCHITECTURE}" == "s390x" || "${ARCHITECTURE}" == "crossbuild-s390x" ]]; then
    default_targets+="
        s390x-guestless-kernel
    "
fi

# Add additional images for x86_64 only
if [[ "${ARCHITECTURE}" != "s390x" && "${ARCHITECTURE}" != "crossbuild-s390x" &&
    "${ARCHITECTURE}" != "aarch64" && "${ARCHITECTURE}" != "crossbuild-aarch64" ]]; then
    default_targets+="
        fedora-realtime-container-disk
        network-passt-binding-cni
        alpine-ext-kernel-boot-demo
    "
fi

# Add additional images for non-s390x architectures (x86_64 + aarch64)
if [[ "${ARCHITECTURE}" != "s390x" && "${ARCHITECTURE}" != "crossbuild-s390x" ]]; then
    default_targets+="
        conformance
        pr-helper
        example-hook-sidecar
        example-disk-mutation-hook-sidecar
        example-cloudinit-hook-sidecar
        example-node-hook-plugin
        test-launcher-hook-sidecar
        cirros-container-disk-demo
        cirros-custom-container-disk-demo
        virtio-container-disk
        winrmcli
        network-passt-binding
        test-network-vhostuser-dra-binding
    "
fi

PUSH_TARGETS=(${PUSH_TARGETS:-${default_targets}})
if [[ -v PUSH_TARGETS_ALT ]]; then
    ALT_TARGETS=(${PUSH_TARGETS_ALT})
else
    ALT_TARGETS=(${PUSH_TARGETS[@]})
fi

if [[ "${BAZEL_IMAGE_REGISTRY:-}" == "1" ]]; then
    if [[ -n "${image_prefix_alt:-}" || -v PUSH_TARGETS_ALT && "${PUSH_TARGETS_ALT}" != "${PUSH_TARGETS[*]}" ]]; then
        echo "Bazel image registry does not support alternate image prefixes or target sets" >&2
        exit 2
    fi
    if [[ "${BUILD_ARCH//,/ }" == *" "* ]]; then
        echo "Bazel image registry currently requires a single build architecture" >&2
        exit 2
    fi

    targets=("${PUSH_TARGETS[@]}" virt-template-apiserver virt-template-controller)
    push_query_targets=()
    for target in "${targets[@]}"; do
        push_query_targets+=("//:push-${target}")
    done
    query_output=$(bazel query --output=build "set(${push_query_targets[*]})")

    image_labels=()
    image_specs=()
    for target in "${targets[@]}"; do
        image_label=$(printf '%s\n' "${query_output}" |
            sed -n "/name = \"push-${target}\"/,/^)/{s/^[[:space:]]*image = \"\([^\"]*\)\".*/\1/p;}" |
            head -n1)
        if [[ -z "${image_label}" ]]; then
            echo "Unable to find image target for //:push-${target}" >&2
            exit 1
        fi
        image_labels+=("${image_label}")
        image_tag=${docker_tag}
        if [[ "${target}" == virt-template-* ]]; then
            image_tag=${VIRT_TEMPLATE_VERSION:-v0.3.0}
        elif [[ -n "${docker_tag_alt}" ]]; then
            image_specs+=("${target}@${docker_tag_alt}=${image_label}")
        fi
        image_specs+=("${target}@${image_tag}=${image_label}")
    done

    bazel build \
        --config="${ARCHITECTURE}" ${BAZEL_CS_CONFIG} \
        --output_groups=oci_tarball \
        --define container_prefix= \
        --define image_prefix= \
        --define container_tag= \
        "${image_labels[@]}"

    registry_args=(
        --listen-address "${BAZEL_REGISTRY_LISTEN_ADDRESS:-0.0.0.0}"
        --port "${BAZEL_REGISTRY_PORT:-5001}"
        --tag "${docker_tag}"
        --repository-prefix "${BAZEL_REGISTRY_REPOSITORY_PREFIX:-kubevirt/${image_prefix:-}}"
        --advertise "${BAZEL_REGISTRY_ADVERTISE:-127.0.0.1}"
    )
    for spec in "${image_specs[@]}"; do
        target=${spec%%@*}
        image_tag=${spec#*@}
        image_tag=${image_tag%%=*}
        image_label=${spec#*=}
        label_path=${image_label#//}
        package=${label_path%%:*}
        image_target=${label_path##*:}
        layout="bazel-bin/${package}/${image_target}_oci_layout.tar"
        registry_args+=(--image "${target}@${image_tag}=${layout}")
    done

    pid_file="${TMPDIR:-/tmp}/kubevirt-bazel-image-registry.pid"
    log_file="${TMPDIR:-/tmp}/kubevirt-bazel-image-registry.log"
    if [[ -f "${pid_file}" ]] && kill -0 "$(cat "${pid_file}")" 2>/dev/null; then
        old_registry_pid=$(cat "${pid_file}")
        kill -TERM "${old_registry_pid}"
        for _ in $(seq 1 30); do
            kill -0 "${old_registry_pid}" 2>/dev/null || break
            sleep 0.1
        done
        if kill -0 "${old_registry_pid}" 2>/dev/null; then
            echo "Bazel image registry process ${old_registry_pid} did not stop" >&2
            exit 1
        fi
    fi
    rm -f "${pid_file}"
    nohup python3 -u hack/bazel_image_registry.py "${registry_args[@]}" >"${log_file}" 2>&1 </dev/null &
    registry_pid=$!
    echo "${registry_pid}" >"${pid_file}"
    sleep 1
    if ! kill -0 "${registry_pid}" 2>/dev/null; then
        cat "${log_file}" >&2
        rm -f "${pid_file}"
        exit 1
    fi
    cat "${log_file}"
    exit 0
fi

for tag in ${docker_tag} ${docker_tag_alt}; do
    if [[ ${tag} == "${docker_tag}" ]]; then
        targets=(${PUSH_TARGETS[@]})
    else
        targets=(${ALT_TARGETS[@]})
    fi
    for target in ${targets[@]}; do

        bazel run \
            --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
            //:push-${target} -- --registry ${docker_prefix} --repository ${image_prefix}${target} --tag ${tag}

    done
done

# for the imagePrefix operator test
if [[ $image_prefix_alt ]]; then
    for target in ${ALT_TARGETS[@]}; do

        bazel run \
            --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
            //:push-${target} -- --registry ${docker_prefix} --repository ${image_prefix_alt}${target} --tag ${docker_tag}

    done
fi

rm -rf ${DIGESTS_DIR}/${ARCHITECTURE}
mkdir -p ${DIGESTS_DIR}/${ARCHITECTURE}

for target in ${PUSH_TARGETS[@]}; do
    dir=${DIGESTS_DIR}/${ARCHITECTURE}/${target}
    mkdir -p ${dir}
    touch ${dir}/${target}.image
done
