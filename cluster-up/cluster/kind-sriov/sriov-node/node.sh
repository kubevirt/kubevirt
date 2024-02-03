#!/bin/bash

SCRIPT_PATH=${SCRIPT_PATH:-$(dirname "$(realpath "$0")")}

CONFIGURE_VFS_SCRIPT_PATH="${SCRIPT_PATH}/configure_vfs.sh"
PFS_IN_USE=${PFS_IN_USE:-}

function node::discover_host_pfs() {
  local -r sriov_pfs=( $(find /sys/class/net/*/device/sriov_numvfs) )
  [ "${#sriov_pfs[@]}" -eq 0 ] && echo "FATAL: Could not find available sriov PFs on host" >&2 && return 1

  local pf_name
  local pf_names=()
  for pf in "${sriov_pfs[@]}"; do
    pf_name="${pf%%/device/*}"
    pf_name="${pf_name##*/}"
    if [ $(echo "${PF_BLACKLIST[@]}" | grep "${pf_name}") ]; then
      continue
    fi

    pfs_names+=( $pf_name )
  done

  echo "${pfs_names[@]}"
}

# node::configure_sriov_pfs moves SR-IOV PFs to nodes netns.
# It exports 'PFS_IN_USE' environment variable with a list
# of SR-IOV PFs that been moved to the nodes netns.
function node::configure_sriov_pfs() {
  local -r nodes_array=($1)
  local -r pfs_names_array=($2)
  local -r pf_count_per_node=$3
  local -r pfs_in_use_var_name=$4

  local pfs_to_move=()
  local pfs_array_offset=0
  local pfs_in_use=()
  local node_exec

  # 'iplink' learns which network namespaces there are by checking /var/run/netns
  mkdir -p /var/run/netns
  for node in "${nodes_array[@]}"; do
    prepare_node_netns "$node"

    ## Move PF's to node netns
    # Slice '$pfs_names_array' to have unique silce for each node
    # with '$pf_count_per_node' PF's names
    pfs_to_move=( "${pfs_names_array[@]:$pfs_array_offset:$pf_count_per_node}" )
    echo "Moving '${pfs_to_move[*]}' PF's to '$node' netns"
    for pf_name in "${pfs_to_move[@]}"; do
      move_pf_to_node_netns "$node" "$pf_name"
    done
    # Increment the offset for next slice
    pfs_array_offset=$((pfs_array_offset + pf_count_per_node))
    pfs_in_use+=( $pf_name )

    # KIND mounts sysfs as read-only by default, remount as R/W"
    node_exec="${CRI_BIN} exec $node"
    $node_exec mount -o remount,rw /sys

    ls_node_dev_vfio="${node_exec} ls -la -Z /dev/vfio"
    $ls_node_dev_vfio
    $node_exec chmod 0666 /dev/vfio/vfio
    $ls_node_dev_vfio

    _kubectl label node $node $SRIOV_NODE_LABEL
  done

  # Set new variable with the used PF names that will consumed by the caller
  eval $pfs_in_use_var_name="'${pfs_in_use[*]}'"
}

# node::configure_sriov_vfs create SR-IOV VFs and configure their driver on each node.
function node::configure_sriov_vfs() {
    local -r nodes_array=($1)
    local -r driver=$2
    local -r driver_kmodule=$3
    local -r vfs_count=$4

    local -r config_vf_script=$(basename "$CONFIGURE_VFS_SCRIPT_PATH")

    for node in "${nodes_array[@]}"; do
      ${CRI_BIN} cp "$CONFIGURE_VFS_SCRIPT_PATH" "$node:/"
      ${CRI_BIN} exec "$node" bash -c "DRIVER=$driver DRIVER_KMODULE=$driver_kmodule VFS_COUNT=$vfs_count ./$config_vf_script"
      ${CRI_BIN} exec "$node" ls -la -Z /dev/vfio
    done
}

function prepare_node_netns() {
  local -r node_name=$1
  local -r node_pid=$($CRI_BIN inspect -f '{{.State.Pid}}' "$node_name")

  # Docker does not create the required symlink for a container netns
  # it perverts iplink from learning that container netns.
  # Thus it is necessary to create symlink between the current
  # worker node (container) netns to /var/run/netns (consumed by iplink)
  # Now the node container netns named with the node name will be visible.
  ln -sf "/proc/$node_pid/ns/net" "/var/run/netns/$node_name"
}

function move_pf_to_node_netns() {
  local -r node_name=$1
  local -r pf_name=$2

  # Move PF to node network-namespace
  ip link set "$pf_name" netns "$node_name"
  # Ensure current PF is up
  ip netns exec "$node_name" ip link set up dev "$pf_name"
  ip netns exec "$node_name" ip link show
}

function node::total_vfs_count() {
  local -r node_name=$1
  local -r node_pid=$($CRI_BIN inspect -f '{{.State.Pid}}' "$node_name")
  local -r pfs_sriov_numvfs=( $(cat /proc/$node_pid/root/sys/class/net/*/device/sriov_numvfs) )
  local total_vfs_on_node=0

  for num_vfs in "${pfs_sriov_numvfs[@]}"; do
    total_vfs_on_node=$((total_vfs_on_node + num_vfs))
  done

  echo "$total_vfs_on_node"
}
