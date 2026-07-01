#!/bin/bash

set -xe

SCRIPT_PATH=$(dirname "$(realpath "$0")")
CONFIGURE_VFS_SCRIPT_PATH="$SCRIPT_PATH/sriov-node/configure_vfs.sh"

# These are expected to be exported by cluster-up/hack/common.sh.
: "${KUBEVIRTCI_CONFIG_PATH:?FATAL: missing KUBEVIRTCI_CONFIG_PATH}"
: "${KUBEVIRT_PROVIDER:?FATAL: missing KUBEVIRT_PROVIDER}"

KUBECONFIG="${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/.kubeconfig"
KUBECTL="${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/.kubectl --kubeconfig=${KUBECONFIG}"

# Configuration
SRIOV_NODE_LABEL_KEY="sriov_capable"
SRIOV_NODE_LABEL_VALUE="true"
SRIOV_NODE_LABEL="$SRIOV_NODE_LABEL_KEY=$SRIOV_NODE_LABEL_VALUE"
SRIOVDP_RESOURCE_PREFIX="kubevirt.io"
SRIOVDP_RESOURCE_NAME="sriov_net"
VFS_DRIVER="vfio-pci"
VFS_DRIVER_KMODULE="vfio_pci"
VFS_COUNT="${VFS_COUNT:-7}"
KUBEVIRT_USE_DRA=${KUBEVIRT_USE_DRA:-false}

# Source SR-IOV components deployment functions
source "${SCRIPT_PATH}/sriov-components/sriov_components.sh"

# SSH function to execute commands on nodes
function ssh_to_node() {
  local node_name=$1
  shift
  "${SCRIPT_PATH}/../../ssh.sh" "$node_name" "$@"
}

# Configure VFs on a single node
function configure_node_vfs() {
  local node_name=$1

  echo "===== Configuring SR-IOV on $node_name ====="

  # Copy configure script to node using base64 encoding
  echo "Copying configure_vfs.sh to $node_name..."
  local encoded_script=$(base64 -w 0 "$CONFIGURE_VFS_SCRIPT_PATH")
  ssh_to_node "$node_name" "echo '$encoded_script' | base64 -d > /tmp/configure_vfs.sh && chmod +x /tmp/configure_vfs.sh"

  # Run configure script on node
  echo "Running VF configuration on $node_name..."
  ssh_to_node "$node_name" "sudo DRIVER=$VFS_DRIVER DRIVER_KMODULE=$VFS_DRIVER_KMODULE VFS_COUNT=$VFS_COUNT bash /tmp/configure_vfs.sh"

  # Label the node
  _kubectl label node "$node_name" "$SRIOV_NODE_LABEL" --overwrite

  echo "===== SR-IOV configuration completed on $node_name ====="
}

# Get total VFs count on a node
function get_node_vfs_count() {
  local node_name=$1
  ssh_to_node "$node_name" "cat /sys/class/net/*/device/sriov_numvfs 2>/dev/null | awk '{s+=\$1} END {print s}'" 2>/dev/null | tr -d '\r' || echo "0"
}

# Wait for allocatable resources to appear
function wait_allocatable_resource() {
  local node_name=$1
  local expected_value=$2

  sriov_components::wait_allocatable_resource "$node_name" "$SRIOVDP_RESOURCE_PREFIX/$SRIOVDP_RESOURCE_NAME" "$expected_value"
}

# Main execution
echo "===== Starting SR-IOV Cluster Configuration ====="

# Get worker nodes
worker_nodes=$(_kubectl get nodes -l node-role.kubernetes.io/worker -o custom-columns=:.metadata.name --no-headers)
worker_nodes_array=($worker_nodes)
worker_nodes_count=${#worker_nodes_array[@]}

if [ "$worker_nodes_count" -eq 0 ]; then
  echo "FATAL: no worker nodes found" >&2
  exit 1
fi

echo "Found $worker_nodes_count worker node(s): ${worker_nodes_array[*]}"

# Configure VFs on each worker node
for node in "${worker_nodes_array[@]}"; do
  configure_node_vfs "$node"
done

echo "===== SR-IOV VF Configuration Complete ====="
echo ""
echo "Node SR-IOV Status:"
for node in "${worker_nodes_array[@]}"; do
  vf_count=$(get_node_vfs_count "$node")
  echo "  $node: $vf_count VFs configured"
done

echo ""
# Collect PF names from all nodes
PFS_IN_USE="sriov0"  # Our SR-IOV interface

if [[ "$KUBEVIRT_USE_DRA" != "true" ]]; then
  echo ""
  echo "===== Deploying SR-IOV Device Plugin ====="
  sriov_components::deploy \
    "$PFS_IN_USE" \
    "$VFS_DRIVER" \
    "$SRIOVDP_RESOURCE_PREFIX" "$SRIOVDP_RESOURCE_NAME" \
    "$SRIOV_NODE_LABEL_KEY" "$SRIOV_NODE_LABEL_VALUE"

  echo ""
  echo "===== Waiting for Allocatable Resources ====="
  # Verify that each sriov capable node has sriov VFs allocatable resource
  for node in "${worker_nodes_array[@]}"; do
    wait_allocatable_resource "$node" "$VFS_COUNT" || exit 1
  done
else
  echo ""
  echo "===== Deploying SR-IOV DRA Driver ====="
  sriov_components::deploy_dra
fi

echo ""
echo "===== Waiting for All Pods to be Ready ====="
sriov_components::wait_pods_ready

echo ""
echo "===== SR-IOV Cluster Configuration Complete ====="
_kubectl get nodes -l "$SRIOV_NODE_LABEL"
_kubectl get pods -n sriov 2>/dev/null || echo "SR-IOV namespace not found (expected if using DRA)"
