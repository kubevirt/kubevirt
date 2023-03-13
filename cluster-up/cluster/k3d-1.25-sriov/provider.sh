#!/usr/bin/env bash

set -e

export CLUSTER_NAME="sriov"
export HOST_PORT=5000

DEPLOY_SRIOV=${DEPLOY_SRIOV:-true}

function print_available_nics() {
    echo 'STEP: Available NICs'
    # print hardware info for easier debugging based on logs
    ${CRI_BIN} run --rm --cap-add=SYS_RAWIO quay.io/phoracek/lspci@sha256:0f3cacf7098202ef284308c64e3fc0ba441871a846022bb87d65ff130c79adb1 sh -c "lspci | egrep -i 'network|ethernet'"
    echo
}

function print_agents_sriov_status() {
    nodes=$(_get_agent_nodes)
    echo "STEP: Print agents SR-IOV status"
    for node in $nodes; do
        echo "Node: $node"
        echo "VFs:"
        ${CRI_BIN} exec $node /bin/sh -c "ls -l /sys/class/net/*/device/virtfn*"
        echo "PFs PCI Addresses:"
        ${CRI_BIN} exec $node /bin/sh -c "grep PCI_SLOT_NAME /sys/class/net/*/device/uevent"
    done
    echo
}

function deploy_sriov() {
    print_available_nics
    ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/config_sriov_cluster.sh
    print_agents_sriov_status
}

function up() {
    k3d_up
    [ $DEPLOY_SRIOV == true ] && deploy_sriov

    version=$(_kubectl get node k3d-$CLUSTER_NAME-server-0 -o=custom-columns=VERSION:.status.nodeInfo.kubeletVersion --no-headers)
    echo "$KUBEVIRT_PROVIDER cluster '$CLUSTER_NAME' is ready ($version)"
}

source ${KUBEVIRTCI_PATH}/cluster/k3d/common.sh
