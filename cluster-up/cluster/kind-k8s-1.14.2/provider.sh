#!/usr/bin/env bash

set -e

export CLUSTER_NAME="kind-1.14.2"
export KIND_NODE_IMAGE="kindest/node:v1.14.2"

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

function up() {
    cp $KIND_MANIFESTS_DIR/kind.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
    kind_up
}
