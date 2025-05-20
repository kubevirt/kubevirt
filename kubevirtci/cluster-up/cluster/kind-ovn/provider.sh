#!/bin/bash -ex
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

KIND_VERSION=0.27.0
export KIND_IMAGE=kindest/node
#export K8S_VERSION=v1.28.0@sha256:dad5a6238c5e41d7cac405fae3b5eda2ad1de6f1190fa8bfc64ff5bb86173213
export K8S_VERSION=v1.32.3

KIND_PATH=${KIND_PATH:-"${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/_kind"}
CLUSTER_PATH=${CLUSTER_PATH:-"${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/_ovnk"}
CLUSTER_NAME=${KUBEVIRT_PROVIDER}

function calculate_mtu() {
    overlay_overhead=58
    current_mtu=$(cat /sys/class/net/$(ip route | grep "default via" | head -1 | awk '{print $5}')/mtu)
    expr $current_mtu - $overlay_overhead
}

MTU=${MTU:-$(calculate_mtu)}

PLATFORM=$(uname -m)
case ${PLATFORM} in
x86_64* | i?86_64* | amd64*)
    ARCH="amd64"
    ;;
aarch64* | arm64*)
    ARCH="arm64"
    ;;
*)
    echo "invalid Arch, only support x86_64, aarch64"
    exit 1
    ;;
esac

function fetch_kind() {
    mkdir -p $KIND_PATH
    current_kind_version=$($KIND_PATH/kind --version |& awk '{print $3}')
    if [[ $current_kind_version != $KIND_VERSION ]]; then
        echo "Downloading kind v$KIND_VERSION"
        curl -LSs https://github.com/kubernetes-sigs/kind/releases/download/v$KIND_VERSION/kind-linux-${ARCH} -o "$KIND_PATH/kind"
        chmod +x "$KIND_PATH/kind"
    fi
    export PATH=$KIND_PATH:$PATH
}

function prepare_config() {
    echo "STEP: Prepare provider config"
    cat >$KUBEVIRTCI_CONFIG_PATH/$KUBEVIRT_PROVIDER/config-provider-$KUBEVIRT_PROVIDER.sh <<EOF
master_ip=127.0.0.1
kubeconfig=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
kubectl=kubectl
docker_prefix=127.0.0.1:5000/kubevirt
manifest_docker_prefix=localhost:5000/kubevirt
EOF
}

function up() {
    cluster::install
    fetch_kind
    pushd $CLUSTER_PATH/contrib ; ./kind.sh --cluster-name $CLUSTER_NAME  --network-segmentation-enable --multi-network-enable -ep podman -mtu $MTU --local-kind-registry --enable-interconnect; popd

    #cp ~/$CLUSTER_NAME.conf "${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig"
    cp ~/ovn.conf "${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig"
    prepare_config
}

function down() {
    ${KIND_PATH}/kind delete cluster --name $CLUSTER_NAME || [[ "$CI" == "true" ]]
}

function _kubectl() {
    export KUBECONFIG=$(${KUBEVIRTCI_PATH}/kubeconfig.sh)
    kubectl "$@"
}

source ${KUBEVIRTCI_PATH}cluster/kind-ovn/install-ovn.sh
