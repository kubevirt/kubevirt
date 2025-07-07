#!/bin/bash

[ $(id -u) -ne 0 ] && echo "FATAL: this script requires sudo privileges" >&2 && exit 1

set -xe

PF_COUNT_PER_NODE=${PF_COUNT_PER_NODE:-1}
[ $PF_COUNT_PER_NODE -le 0 ] && echo "FATAL: PF_COUNT_PER_NODE must be a positive integer" >&2 && exit 1

SCRIPT_PATH=$(dirname "$(realpath "$0")")

source ${SCRIPT_PATH}/sriov-node/node.sh
source ${SCRIPT_PATH}/sriov-components/sriov_components.sh

CONFIGURE_VFS_SCRIPT_PATH="$SCRIPT_PATH/sriov-node/configure_vfs.sh"

SRIOV_COMPONENTS_NAMESPACE="sriov"
SRIOV_NODE_LABEL_KEY="sriov_capable"
SRIOV_NODE_LABEL_VALUE="true"
SRIOV_NODE_LABEL="$SRIOV_NODE_LABEL_KEY=$SRIOV_NODE_LABEL_VALUE"
SRIOVDP_RESOURCE_PREFIX="kubevirt.io"
SRIOVDP_RESOURCE_NAME="sriov_net"
VFS_DRIVER="vfio-pci"
VFS_DRIVER_KMODULE="vfio_pci"
VFS_COUNT="6"

function validate_nodes_sriov_allocatable_resource() {
  local -r resource_name="$SRIOVDP_RESOURCE_PREFIX/$SRIOVDP_RESOURCE_NAME"
  local -r sriov_nodes=$(_kubectl get nodes -l $SRIOV_NODE_LABEL -o custom-columns=:.metadata.name --no-headers)

  local num_vfs
  for sriov_node in $sriov_nodes; do
    num_vfs=$(node::total_vfs_count "$sriov_node")
    sriov_components::wait_allocatable_resource "$sriov_node" "$resource_name" "$num_vfs"
  done
}

worker_nodes=($(_kubectl get nodes -l node-role.kubernetes.io/worker -o custom-columns=:.metadata.name --no-headers))
worker_nodes_count=${#worker_nodes[@]}
[ "$worker_nodes_count" -eq 0 ] && echo "FATAL: no worker nodes found" >&2 && exit 1

pfs_names=($(node::discover_host_pfs))
pf_count="${#pfs_names[@]}"
[ "$pf_count" -eq 0 ] && echo "FATAL: Could not find available sriov PF's" >&2 && exit 1

total_pf_required=$((worker_nodes_count*PF_COUNT_PER_NODE))
[ "$pf_count" -lt "$total_pf_required" ] && \
  echo "FATAL: there are not enough PF's on the host, try to reduce PF_COUNT_PER_NODE
  Worker nodes count: $worker_nodes_count
  PF per node count:  $PF_COUNT_PER_NODE
  Total PF count required:  $total_pf_required" >&2 && exit 1

## Move SR-IOV Physical Functions to worker nodes
PFS_IN_USE=""
node::configure_sriov_pfs "${worker_nodes[*]}" "${pfs_names[*]}" "$PF_COUNT_PER_NODE" "PFS_IN_USE"

## Create VFs and configure their drivers on each SR-IOV node
node::configure_sriov_vfs "${worker_nodes[*]}" "$VFS_DRIVER" "$VFS_DRIVER_KMODULE" "$VFS_COUNT"

## Deploy Multus and SRIOV components
sriov_components::deploy_multus
sriov_components::deploy \
  "$PFS_IN_USE" \
  "$VFS_DRIVER" \
  "$SRIOVDP_RESOURCE_PREFIX" "$SRIOVDP_RESOURCE_NAME" \
  "$SRIOV_NODE_LABEL_KEY" "$SRIOV_NODE_LABEL_VALUE"

# Verify that each sriov capable node has sriov VFs allocatable resource
validate_nodes_sriov_allocatable_resource
sriov_components::wait_pods_ready

_kubectl get nodes
_kubectl get pods -n $SRIOV_COMPONENTS_NAMESPACE
