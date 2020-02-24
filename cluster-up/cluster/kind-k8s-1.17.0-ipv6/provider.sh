#!/usr/bin/env bash

set -e

export IPV6_CNI="yes"
export CLUSTER_NAME="kind-1.17.0"
export KIND_NODE_IMAGE="kindest/node:v1.17.0"

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

function up() {
    cp $KIND_MANIFESTS_DIR/kind-ipv6.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
    kind_up
}
