# Copyright 2018-2019 Red Hat, Inc.
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

export KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-'k8s-1.23'}
export KUBEVIRTCI_TAG=$(curl -L -Ss https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest)
KUBEVIRTCI_PATH="${PWD}/_kubevirtci"
KUBEVIRTCI_REPO='https://github.com/kubevirt/kubevirtci.git'

function cluster::_get_repo() {
    git --git-dir ${KUBEVIRTCI_PATH}/.git remote get-url origin
}

function cluster::_get_tag() {
    git -C ${KUBEVIRTCI_PATH} describe --tags
}

function kubevirtci::install() {
    # Remove cloned kubevirtci repository if it does not match the requested one
    if [ -d ${KUBEVIRTCI_PATH} ]; then
        if [ $(cluster::_get_repo) != ${KUBEVIRTCI_REPO} ] || [ $(cluster::_get_tag) != ${KUBEVIRTCI_TAG} ]; then
            rm -rf ${KUBEVIRTCI_PATH}
        fi
    fi

    if [ ! -d ${KUBEVIRTCI_PATH} ]; then
        git clone https://github.com/kubevirt/kubevirtci.git ${KUBEVIRTCI_PATH}
        (
            cd ${KUBEVIRTCI_PATH}
            git checkout tags/${KUBEVIRTCI_TAG} -b ${KUBEVIRTCI_TAG}
        )
    fi
}

function kubevirtci::path() {
    echo -n ${KUBEVIRTCI_PATH}
}

function kubevirtci::kubeconfig() {
    echo -n ${KUBEVIRTCI_PATH}/_ci-configs/${KUBEVIRT_PROVIDER}/.kubeconfig
}
