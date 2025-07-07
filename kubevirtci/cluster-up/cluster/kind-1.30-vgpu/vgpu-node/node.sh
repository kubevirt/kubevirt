#!/bin/bash

function node::discover_host_gpus() {
  local -r gpu_types=( $(find /sys/class/mdev_bus/*/mdev_supported_types) )
  [ "${#gpu_types[@]}" -eq 0 ] && echo "FATAL: Could not find available GPUs on host" >&2 && return 1

  local gpu_addr
  local gpu_addresses=()
  for path in "${gpu_types}"; do
    gpu_addr="${gpu_types#/sys/class/mdev_bus/}"
    gpu_addr=${gpu_addr%/*}

    gpu_addresses+=( $gpu_addr )
  done

  echo "${gpu_addresses[@]}"
}

function node::remount_sysfs() {
  local -r nodes_array=($1)
  local node_exec

  for node in "${nodes_array[@]}"; do

    # KIND mounts sysfs as read-only by default, remount as R/W"
    node_exec="${CRI_BIN} exec $node"
    $node_exec mount -o remount,rw /sys
    $node_exec chmod 666 /dev/vfio/vfio

  done
}

