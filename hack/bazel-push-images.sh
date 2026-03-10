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
    alpine-with-test-tooling-container-disk
    fedora-with-test-tooling-container-disk
    vm-killer
    sidecar-shim
    disks-images-provider
    libguestfs-tools
    test-helpers
"

# Add additional images for s390x only
if [[ "${ARCHITECTURE}" == "s390x" || "${ARCHITECTURE}" == "crossbuild-s390x" ]]; then
    default_targets+="
        s390x-guestless-kernel
    "
fi

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
        fedora-realtime-container-disk
        winrmcli
        network-slirp-binding
        network-passt-binding
        network-passt-binding-cni
    "
fi

PUSH_TARGETS=(${PUSH_TARGETS:-${default_targets}})
KUBEVIRT_PUSH_PARALLELISM=${KUBEVIRT_PUSH_PARALLELISM:-1}

if ! [[ "${KUBEVIRT_PUSH_PARALLELISM}" =~ ^[0-9]+$ ]] || [[ "${KUBEVIRT_PUSH_PARALLELISM}" -lt 1 ]]; then
    echo "KUBEVIRT_PUSH_PARALLELISM must be a positive integer, got '${KUBEVIRT_PUSH_PARALLELISM}'"
    exit 1
fi

function push_image() {
    local target=$1
    local repository=$2
    local tag=$3

    bazel run \
        --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
        //:push-${target} -- --repository ${repository} --tag ${tag}
}

function push_images_sequentially() {
    for tag in ${docker_tag} ${docker_tag_alt}; do
        for target in ${PUSH_TARGETS[@]}; do
            push_image ${target} ${docker_prefix}/${image_prefix}${target} ${tag}
        done
    done

    # for the imagePrefix operator test
    if [[ $image_prefix_alt ]]; then
        for target in ${PUSH_TARGETS[@]}; do
            push_image ${target} ${docker_prefix}/${image_prefix_alt}${target} ${docker_tag}
        done
    fi
}

function push_images_in_parallel() {
    local log_dir
    log_dir=$(mktemp -d)
    trap 'rm -rf "${log_dir}"' EXIT

    local -a task_targets=()
    local -a task_repositories=()
    local -a task_tags=()
    local -a task_logs=()
    local -a task_statuses=()

    local task_count=0

    function enqueue_task() {
        local target=$1
        local repository=$2
        local tag=$3

        task_targets[${task_count}]="${target}"
        task_repositories[${task_count}]="${repository}"
        task_tags[${task_count}]="${tag}"
        task_logs[${task_count}]="${log_dir}/${task_count}-${target}-${tag}.log"
        task_statuses[${task_count}]="1"
        task_count=$((task_count + 1))
    }

    for tag in ${docker_tag} ${docker_tag_alt}; do
        for target in ${PUSH_TARGETS[@]}; do
            enqueue_task ${target} ${docker_prefix}/${image_prefix}${target} ${tag}
        done
    done

    # for the imagePrefix operator test
    if [[ $image_prefix_alt ]]; then
        for target in ${PUSH_TARGETS[@]}; do
            enqueue_task ${target} ${docker_prefix}/${image_prefix_alt}${target} ${docker_tag}
        done
    fi

    local -a running_pids=()
    local -a running_task_ids=()

    function wait_for_task() {
        local pid=$1
        local task_id=$2

        if wait ${pid}; then
            task_statuses[${task_id}]="0"
        else
            task_statuses[${task_id}]="$?"
        fi
    }

    function wait_for_oldest_running_task() {
        wait_for_task ${running_pids[0]} ${running_task_ids[0]}
        running_pids=("${running_pids[@]:1}")
        running_task_ids=("${running_task_ids[@]:1}")
    }

    local task_id
    for ((task_id = 0; task_id < task_count; task_id++)); do
        push_image \
            "${task_targets[${task_id}]}" \
            "${task_repositories[${task_id}]}" \
            "${task_tags[${task_id}]}" \
            >"${task_logs[${task_id}]}" 2>&1 &

        running_pids+=("$!")
        running_task_ids+=("${task_id}")

        if [[ ${#running_pids[@]} -ge ${KUBEVIRT_PUSH_PARALLELISM} ]]; then
            wait_for_oldest_running_task
        fi
    done

    while [[ ${#running_pids[@]} -gt 0 ]]; do
        wait_for_oldest_running_task
    done

    local failures=0
    for ((task_id = 0; task_id < task_count; task_id++)); do
        local target=${task_targets[${task_id}]}
        local repository=${task_repositories[${task_id}]}
        local tag=${task_tags[${task_id}]}
        local status=${task_statuses[${task_id}]}

        echo "[PUSH][${task_id}/${task_count}] ${repository}:${tag} (target=${target})"
        cat "${task_logs[${task_id}]}"

        if [[ ${status} != "0" ]]; then
            echo "[PUSH][FAILED] ${repository}:${tag} (target=${target}, exit=${status})"
            failures=$((failures + 1))
        fi
    done

    if [[ ${failures} -gt 0 ]]; then
        echo "${failures} image push task(s) failed"
        exit 1
    fi
}

if [[ ${KUBEVIRT_PUSH_PARALLELISM} -eq 1 ]]; then
    push_images_sequentially
else
    push_images_in_parallel
fi

rm -rf ${DIGESTS_DIR}/${ARCHITECTURE}
mkdir -p ${DIGESTS_DIR}/${ARCHITECTURE}

for target in ${PUSH_TARGETS[@]}; do
    dir=${DIGESTS_DIR}/${ARCHITECTURE}/${target}
    mkdir -p ${dir}
    touch ${dir}/${target}.image
done
