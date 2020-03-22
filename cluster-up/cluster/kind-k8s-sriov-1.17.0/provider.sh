#!/usr/bin/env bash

set -e

export CLUSTER_NAME="sriov"
export KIND_NODE_IMAGE="kindest/node:v1.17.0"

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

function up() {
    if [[ "$KUBEVIRT_NUM_NODES" -ne 2 ]]; then
        echo 'SR-IOV cluster can be only started with 2 nodes'
        exit 1
    fi

    # print hardware info for easier debugging based on logs
    echo 'Available NICs'
    docker run --rm --cap-add=SYS_RAWIO quay.io/phoracek/lspci@sha256:0f3cacf7098202ef284308c64e3fc0ba441871a846022bb87d65ff130c79adb1 sh -c "lspci | egrep -i 'network|ethernet'"

    cp $KIND_MANIFESTS_DIR/kind.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
    _fetch_kind
    prepare_workers
    # adding mounts to control plane, need them for sriov
    cat >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml << EOF 
  extraMounts:
  - containerPath: /lib/modules
    hostPath: /lib/modules
    readOnly: true
  - containerPath: /dev/vfio/
    hostPath: /dev/vfio/
EOF

    setup_kind
    ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/config_sriov.sh
}
