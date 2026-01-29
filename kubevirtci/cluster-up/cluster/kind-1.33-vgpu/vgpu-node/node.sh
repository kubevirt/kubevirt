#!/bin/bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright the KubeVirt Authors.

function node::discover_host_gpus() {
  local -r gpu_types=( $(find /sys/class/mdev_bus/*/mdev_supported_types 2>/dev/null) )
  
  if [ "${#gpu_types[@]}" -eq 0 ]; then
    echo "WARNING: Could not find available GPUs on host" >&2
    echo "If using fake vGPU, ensure the module is loaded" >&2
    return 0
  fi

  local gpu_addr
  local gpu_addresses=()
  for path in "${gpu_types[@]}"; do
    # Extract parent device name from path
    # e.g., /sys/class/mdev_bus/fake_nvidia_vgpu/mdev_supported_types -> fake_nvidia_vgpu
    gpu_addr="${path#/sys/class/mdev_bus/}"
    gpu_addr="${gpu_addr%%/*}"

    gpu_addresses+=( "$gpu_addr" )
  done

  echo "Found GPU/mdev devices: ${gpu_addresses[*]}"
  echo "${gpu_addresses[@]}"
}

function node::remount_sysfs() {
  local -r nodes_array=($1)
  local node_exec

  for node in "${nodes_array[@]}"; do

    # KIND mounts sysfs as read-only by default, remount as R/W"
    node_exec="${CRI_BIN} exec $node"
    $node_exec mount -o remount,rw /sys 2>/dev/null || true
    $node_exec chmod 666 /dev/vfio/vfio 2>/dev/null || true

  done
}

function node::list_mdev_types() {
  echo "Available mdev types:"
  for parent in /sys/class/mdev_bus/*; do
    if [ -d "$parent/mdev_supported_types" ]; then
      local parent_name=$(basename "$parent")
      echo "  Parent: $parent_name"
      for type_dir in "$parent/mdev_supported_types"/*; do
        if [ -d "$type_dir" ]; then
          local type_name=$(basename "$type_dir")
          local pretty_name=$(cat "$type_dir/name" 2>/dev/null || echo "N/A")
          local available=$(cat "$type_dir/available_instances" 2>/dev/null || echo "N/A")
          echo "    - $type_name: $pretty_name ($available available)"
        fi
      done
    fi
  done
}

function node::list_mdev_devices() {
  echo "Active mdev devices:"
  if [ -d /sys/bus/mdev/devices ]; then
    for dev in /sys/bus/mdev/devices/*; do
      if [ -d "$dev" ]; then
        local uuid=$(basename "$dev")
        local mdev_type=$(basename "$(readlink -f "$dev/mdev_type" 2>/dev/null)" || echo "unknown")
        echo "  $uuid (type: $mdev_type)"
      fi
    done
  else
    echo "  None"
  fi
}

