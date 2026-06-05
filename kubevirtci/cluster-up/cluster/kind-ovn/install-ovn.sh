#!/bin/bash -e
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
# Copyright 2024 Red Hat, Inc.
#

export OVNK_COMMIT=5b4cd1e52958dd47a6880accac2e2aa2a33d4ffc

OVNK_REPO='https://github.com/ovn-org/ovn-kubernetes.git'
CLUSTER_PATH=${CLUSTER_PATH:-"${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/_ovnk"}

function cluster::_get_repo() {
    git --git-dir ${CLUSTER_PATH}/.git config --get remote.origin.url
}

function cluster::_get_sha() {
    git --git-dir ${CLUSTER_PATH}/.git rev-parse HEAD
}

function cluster::install() {
    if [ -d ${CLUSTER_PATH} ]; then
        if [ $(cluster::_get_repo) != ${OVNK_REPO} -o $(cluster::_get_sha) != ${OVNK_COMMIT} ]; then
            rm -rf ${CLUSTER_PATH}
        fi
    fi

    if [ ! -d ${CLUSTER_PATH} ]; then
        git clone ${OVNK_REPO} ${CLUSTER_PATH}
        (
            cd ${CLUSTER_PATH}
            git checkout ${OVNK_COMMIT}
        )
    fi
}
