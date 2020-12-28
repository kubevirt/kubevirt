#!/bin/bash

set -ex

if [ "$(id -u)" -ne 0 ]; then
  echo "This script requires sudo privileges"
  exit 1
fi

source cluster-up/hack/common.sh
source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

function get_pf_names() {
  local pf_net_devices=( $(ls /sys/class/net/*/device/sriov_numvfs) )
  local pf_names=()
  for pf in "${pf_net_devices[@]}"; do
    pf_name="${pf%%/device/*}"
    pf_name="${pf_name##*/}"

    if [ $(echo "${PF_BLACKLIST[@]}" | grep "${pf_name}") ]; then
      continue
    fi

    pf_names+=( $pf_name )
  done

  echo "${pf_names[@]}"
}

function attach_pf_to_network_namespace() {
  local -r current_pf=$1 
  local -r current_node=$2

  echo "[$current_node] Attaching PF '$current_pf' to node network namespace"
  
  pid="$(docker inspect -f '{{.State.Pid}}' $current_node)"
  current_node_network_namespace=$current_node

  # Create symlink to current worker node (container) network-namespace
  # at /var/run/netns (consumned by iplink) so it will be visibale by iplink.
  # This is necessary since docker does not creating the requierd
  # symlink for a container network-namesapce.
  ln -sf /proc/$pid/ns/net "/var/run/netns/$current_node_network_namespace"

  # Move current PF to current node network-namespace
  ip link set $current_pf netns $current_node_network_namespace

  # Ensure current PF is up
  ip netns exec $current_node_network_namespace ip link set up dev $current_pf

  ip netns exec $current_node_network_namespace ip link show
}

CRI=${CRI:-docker}

PF_BLACKLIST=${PF_BLACKLIST:-none}
SRIOV_WORKER_NODES_LABEL=${SRIOV_WORKER_NODES_LABEL:-none}

echo "SRIOV Pysical Functions interfaces names"
pf_names=( $(get_pf_names) )
if [ ${#pf_names[@]} -eq 0 ];then
  echo "No PF's found"
  exit 1
fi
echo "${pf_names[@]}"

echo "Worker nodes"
nodes=( $(_kubectl get nodes -l node-role.kubernetes.io/worker -o custom-columns=:.metadata.name --no-headers) )
if [ ${#nodes[@]} -eq 0 ];then
  echo "No worker nodes found"
  exit 1
fi
echo "${nodes[@]}"

echo "Move PF's to worker nodes network namespaces"
mkdir -p /var/run/netns/
pf_count="${#pf_names[@]}"
for i in $(seq $pf_count); do
  index=$((i-1))

  current_pf="${pf_names[$index]}"
  if [ -z $current_pf ]; then
    echo "All PF's were attached"
    break
  fi

  current_node="${nodes[$index]}"
  if [ -z $current_node ]; then
    echo "All workers were configured"
    break
  fi

  attach_pf_to_network_namespace $current_pf $current_node

  _kubectl label node $current_node $SRIOV_WORKER_NODES_LABEL

  node_exec="$CRI exec $current_node"
  # KIND mounts sysfs as readonly
  ${node_exec} mount -o remount,rw /sys

  # Ensure vfio binary is executable
  ${node_exec} chmod 666 /dev/vfio/vfio

  # Prepare SRIOV Virtual Functions
  ${CRI} cp "${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/configure_vfs.sh" $current_node:/
  ${node_exec} bash -c "./configure_vfs.sh"
done
