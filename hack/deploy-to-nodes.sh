#!/usr/bin/env bash
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


source hack/common.sh
source kubevirtci/cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh

set -e

readonly PODMAN_SOCKET=${PODMAN_SOCKET:-"/run/podman/podman.sock"}

detect_podman_socket() {
    if curl --unix-socket "${PODMAN_SOCKET}" http://d/v3.0.0/libpod/info >/dev/null 2>&1; then
        echo "${PODMAN_SOCKET}"
    fi
}

if [ "${CONTAINER_CLIENT}" = "podman" ]; then
    _cri_bin="podman --remote --url=unix://$(detect_podman_socket)"
elif [ "${CONTAINER_CLIENT}" = "docker" ]; then
    _cri_bin=docker
else
    _cri_socket=$(detect_podman_socket)
    if [ -n "$_cri_socket" ]; then
        _cri_bin="podman --remote --url=unix://$_cri_socket"
        echo >&2 "selecting podman as container runtime"
    elif docker ps >/dev/null 2>&1; then
        _cri_bin=docker
        echo >&2 "selecting docker as container runtime"
    else
        echo >&2 "no working container runtime found. Neither docker nor podman seems to work."
        exit 1
    fi
fi

readonly container_prefix=${JOB_NAME:-${KUBEVIRT_PROVIDER}}

# Upload a file to all cluster nodes
# Arguments:
#   1. file: The file name.
#   2. src_path: The local path to the file.
#   3. dst_path: The target path, on the node, to upload the file to.
#   4. nodes  : The node names on the cluster, to upload the file to.
upload_file_to_node() {
    local -r file=$1
    local -r src_path=$2
    local -r dst_path=$3
    local -r nodes=$4

    for node in $nodes; do
        local node_container="${container_prefix}-${node}"

        echo "Start deploying ${file} to ${node}"
        ${_cri_bin} cp "${src_path}/${file}" "${node_container}":/tmp/
        ${_cri_bin} exec -it "${node_container}" ssh -o "StrictHostKeyChecking no" -i vagrant.key "vagrant@192.168.66.10${node:0-1}" rm -f "/tmp/${file}"
        ${_cri_bin} exec -it "${node_container}" scp -o "StrictHostKeyChecking no" -i vagrant.key "/tmp/${file}" "vagrant@192.168.66.10${node:0-1}":/tmp/
        ${_cri_bin} exec -it "${node_container}" ssh -o "StrictHostKeyChecking no" -i vagrant.key "vagrant@192.168.66.10${node:0-1}" sudo cp "/tmp/${file}" "${dst_path}/"

        echo "${file} deployed successfully on node ${node}"
        ${_cri_bin} exec -it "${node_container}" ssh -o "StrictHostKeyChecking no" -i vagrant.key "vagrant@192.168.66.10${node:0-1}" ls -la "${dst_path}/${file}"
    done
}

readonly cluster_nodes=$(_kubectl get nodes -o custom-columns=:.metadata.name --no-headers)

# Deploy CNI plugin passt
if [ "${KUBEVIRT_DEPLOY_NET_BINDING_CNI}" == "true" ]; then
    upload_file_to_node "kubevirt-passt-binding" "${OUT_DIR}/cmd/cniplugins" "/opt/cni/bin" "${cluster_nodes}"
fi
