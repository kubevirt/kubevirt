#!/bin/bash

[ $(id -u) -ne 0 ] && echo "FATAL: this script requires sudo privileges" >&2 && exit 1

set -xe

PF_COUNT_PER_NODE=${PF_COUNT_PER_NODE:-1}
[ $PF_COUNT_PER_NODE -le 0 ] && echo "FATAL: PF_COUNT_PER_NODE must be a positive integer" >&2 && exit 1

SCRIPT_PATH=$(dirname "$(realpath "$0")")

source ${SCRIPT_PATH}/sriov-components/sriov_components.sh

CONFIGURE_VFS_SCRIPT_PATH="$SCRIPT_PATH/configure_vfs.sh"

MASTER_NODE="${CLUSTER_NAME}-control-plane"
WORKER_NODE_ROOT="${CLUSTER_NAME}-worker"

SRIOV_COMPONENTS_NAMESPACE="sriov"
SRIOV_NODE_LABEL_KEY="sriov_capable"
SRIOV_NODE_LABEL_VALUE="true"
SRIOV_NODE_LABEL="$SRIOV_NODE_LABEL_KEY=$SRIOV_NODE_LABEL_VALUE"
SRIOVDP_RESOURCE_PREFIX="kubevirt.io"
SRIOVDP_RESOURCE_NAME="sriov_net"
VFS_DRIVER="vfio-pci"
VFS_DRIVER_KMODULE="vfio_pci"

function move_sriov_pfs_netns_to_node {
  local -r node=$1
  local -r pf_count_per_node=$2
  local -r pid="$(docker inspect -f '{{.State.Pid}}' $node)"
  local pf_array=()

  mkdir -p /var/run/netns/
  ln -sf /proc/$pid/ns/net "/var/run/netns/$node"

  local -r sriov_pfs=( $(find /sys/class/net/*/device/sriov_numvfs) )
  [ "${#sriov_pfs[@]}" -eq 0 ] && echo "FATAL: Could not find available sriov PFs" >&2 && return 1

  for pf in "${sriov_pfs[@]}"; do
    local pf_name="${pf%%/device/*}"
    pf_name="${pf_name##*/}"

    if [ $(echo "${PF_BLACKLIST[@]}" | grep "${pf_name}") ]; then
      continue
    fi

    # In case two clusters started at the same time, they might race on the same PF.
    # The first will manage to assign the PF to its container, and the 2nd will just skip it
    # and try the rest of the PFs available.
    if ip link set "$pf_name" netns "$node"; then
      if timeout 10s bash -c "until ip netns exec $node ip link show $pf_name > /dev/null; do sleep 1; done"; then
        pf_array+=("$pf_name")
        [ "${#pf_array[@]}" -eq "$pf_count_per_node" ] && break
      fi
    fi
  done

  [ "${#pf_array[@]}" -lt "$pf_count_per_node" ] && \
    echo "FATAL: Not enough PFs allocated, PF_BLACKLIST (${PF_BLACKLIST[@]}), PF_COUNT_PER_NODE ${PF_COUNT_PER_NODE}" >&2 && \
    return 1

  echo "${pf_array[@]}"
}

# The first worker needs to be handled specially as it has no ending number, and sort will not work
# We add the 0 to it and we remove it if it's the candidate worker
WORKER=$(_kubectl get nodes | grep $WORKER_NODE_ROOT | sed "s/\b$WORKER_NODE_ROOT\b/${WORKER_NODE_ROOT}0/g" | sort -r | awk 'NR==1 {print $1}')
if [[ -z "$WORKER" ]]; then
  SRIOV_NODE=$MASTER_NODE
else
  SRIOV_NODE=$WORKER
fi

# this is to remove the ending 0 in case the candidate worker is the first one
if [[ "$SRIOV_NODE" == "${WORKER_NODE_ROOT}0" ]]; then
  SRIOV_NODE=${WORKER_NODE_ROOT}
fi

NODE_PFS=($(move_sriov_pfs_netns_to_node "$SRIOV_NODE" "$PF_COUNT_PER_NODE"))
NODE_PF=$NODE_PFS

SRIOV_NODE_CMD="docker exec ${SRIOV_NODE}"
NODE_PF_NUM_VFS=$(${SRIOV_NODE_CMD} cat /sys/class/net/$NODE_PF/device/sriov_totalvfs)

# KIND mounts sysfs as readonly, this script requires R/W access to sysfs
${SRIOV_NODE_CMD} mount -o remount,rw /sys
${SRIOV_NODE_CMD} chmod 666 /dev/vfio/vfio

# Create and configure SRIOV Virtual Functions on SRIOV node
docker cp "$CONFIGURE_VFS_SCRIPT_PATH" "$SRIOV_NODE:/"
config_vf_script=$(basename "$CONFIGURE_VFS_SCRIPT_PATH")
${SRIOV_NODE_CMD} bash -c "DRIVER=$VFS_DRIVER DRIVER_KMODULE=$VFS_DRIVER_KMODULE ./$config_vf_script"

_kubectl label node $SRIOV_NODE $SRIOV_NODE_LABEL

sriov_components::deploy_multus
sriov_components::deploy \
  "$NODE_PF" \
  "$VFS_DRIVER" \
  "$SRIOVDP_RESOURCE_PREFIX" "$SRIOVDP_RESOURCE_NAME" \
  "$SRIOV_NODE_LABEL_KEY" "$SRIOV_NODE_LABEL_VALUE"

# Verify that sriov node has sriov VFs allocatable resource
resource_name="$SRIOVDP_RESOURCE_PREFIX/$SRIOVDP_RESOURCE_NAME"
sriov_components::wait_allocatable_resource "$SRIOV_NODE" "$resource_name" "$NODE_PF_NUM_VFS"
sriov_components::wait_pods_ready

_kubectl get nodes
_kubectl get pods -n $SRIOV_COMPONENTS_NAMESPACE
