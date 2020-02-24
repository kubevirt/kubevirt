#!/usr/bin/env bash

set -e

export CLUSTER_NAME="kind-sriov-1.14.2"
export KIND_NODE_IMAGE="kindest/node:v1.14.2"

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

function up() {
    cp $KIND_MANIFESTS_DIR/kind.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
    # adding mounts to control plane, need them for sriov
    cat >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml << EOF 
  extraMounts:
  - containerPath: /lib/modules
    hostPath: /lib/modules
    readOnly: true
  - containerPath: /dev/vfio/
    hostPath: /dev/vfio/
EOF

    kind_up
    ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/config_sriov.sh
}
