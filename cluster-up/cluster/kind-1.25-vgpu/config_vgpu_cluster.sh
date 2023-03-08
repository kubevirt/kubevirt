#!/bin/bash

[ $(id -u) -ne 0 ] && echo "FATAL: this script requires sudo privileges" >&2 && exit 1

set -xe

SCRIPT_PATH=$(dirname "$(realpath "$0")")

source ${SCRIPT_PATH}/vgpu-node/node.sh
echo "_kubectl: " ${_kubectl}
echo "KUBECTL_PATH: " $KUBECTL_PATH
echo "KUBEVIRTCI_PATH: " ${KUBEVIRTCI_PATH}
source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh
echo "_kubectl: " ${_kubectl}

nodes=($(_kubectl get nodes -o custom-columns=:.metadata.name --no-headers))
node::remount_sysfs "${nodes[*]}"
node::discover_host_gpus

_kubectl get nodes
