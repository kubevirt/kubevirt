#!/bin/bash -e
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
# Copyright Red Hat, Inc.
#

KIND_VERSION=0.19.0
KIND_PATH=${KIND_PATH:-"${PWD}/_kind"}
CLUSTER_PATH=${CLUSTER_PATH:-"${PWD}/_ovnk"}

PLATFORM=$(uname -m)
case ${PLATFORM} in
x86_64* | i?86_64* | amd64*)
    ARCH="amd64"
    ;;
ppc64le)
    ARCH="ppc64le"
    ;;
aarch64* | arm64*)
    ARCH="arm64"
    ;;
*)
    echo "invalid Arch, only support x86_64, ppc64le, aarch64"
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

source hack/install-ovn.sh
cluster::install

fetch_kind

pushd $CLUSTER_PATH/contrib ; ./kind.sh --multi-network-enable -lr ; popd
export KUBECONFIG=$(realpath ~/ovn.conf)

export DOCKER_PREFIX=localhost:5000
export KUBEVIRT_PROVIDER=external
make cluster-sync
