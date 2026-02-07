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

function node::install_mesa() {
  # Install mesa/OpenGL libraries in Kind nodes for vGPU display support
  local -r nodes_array=($1)
  local node_exec

  echo "Installing mesa/OpenGL libraries in Kind nodes..."

  for node in "${nodes_array[@]}"; do
    node_exec="${CRI_BIN} exec $node"

    echo "  Installing on node: $node"
    # Kind nodes are based on Debian/Ubuntu, use apt
    # Need mesa libraries AND all their dependencies:
    # - libegl-mesa0: provides libEGL_mesa.so.0 (actual mesa EGL implementation)
    # - libglx-mesa0: provides libGLX_mesa.so.0 (actual mesa GLX implementation)
    # - libgbm1: provides libgbm.so.1 (GBM buffer management for DMABUF)
    # - libgl1-mesa-dri: DRI drivers
    # - libglapi-mesa0: libglapi.so.0 (mesa GL API)
    # - libdrm2: libdrm.so.2 (Direct Rendering Manager)
    # - libxcb*, libx11-xcb1: X11/XCB libraries for display
    # - libwayland-*: Wayland libraries
    # - libxshmfence1: X shared memory fence
    $node_exec bash -c 'apt-get update -qq && apt-get install -qq -y \
      libgl1-mesa-dri \
      libegl-mesa0 \
      libglx-mesa0 \
      libgbm1 \
      libgl1 \
      libegl1 \
      libgles2 \
      libglvnd0 \
      libglx0 \
      libglapi-mesa \
      libdrm2 \
      libx11-xcb1 \
      libxcb1 \
      libxcb-dri2-0 \
      libxcb-dri3-0 \
      libxcb-present0 \
      libxcb-sync1 \
      libxcb-xfixes0 \
      libxcb-randr0 \
      libxcb-glx0 \
      libxshmfence1 \
      libwayland-client0 \
      libwayland-server0 \
      libxau6 \
      libxdmcp6 \
      libbsd0 \
      libmd0 \
      >/dev/null 2>&1' || {
        echo "    Warning: Failed to install mesa packages on $node (may already exist or not supported)"
      }

    # Create symlinks from RHEL-style paths (/usr/lib64/) to Debian paths (/usr/lib/x86_64-linux-gnu/)
    # This is needed because the mesa-injector webhook mounts from /usr/lib64/ paths,
    # but Kind nodes use Debian which installs libraries to /usr/lib/x86_64-linux-gnu/
    echo "  Creating symlinks for RHEL-style paths on node: $node"
    $node_exec bash -c '
      mkdir -p /usr/lib64
      DEBLIB=/usr/lib/x86_64-linux-gnu
      
      # Mesa implementations
      ln -sf $DEBLIB/libEGL_mesa.so.0 /usr/lib64/libEGL_mesa.so.0
      ln -sf $DEBLIB/libGLX_mesa.so.0 /usr/lib64/libGLX_mesa.so.0 2>/dev/null || true
      ln -sf $DEBLIB/libgbm.so.1 /usr/lib64/libgbm.so.1
      ln -sf $DEBLIB/libglapi.so.0 /usr/lib64/libglapi.so.0
      
      # DRM
      ln -sf $DEBLIB/libdrm.so.2 /usr/lib64/libdrm.so.2
      
      # X11/XCB libraries
      ln -sf $DEBLIB/libX11-xcb.so.1 /usr/lib64/libX11-xcb.so.1
      ln -sf $DEBLIB/libxcb.so.1 /usr/lib64/libxcb.so.1
      ln -sf $DEBLIB/libxcb-dri2.so.0 /usr/lib64/libxcb-dri2.so.0
      ln -sf $DEBLIB/libxcb-dri3.so.0 /usr/lib64/libxcb-dri3.so.0
      ln -sf $DEBLIB/libxcb-present.so.0 /usr/lib64/libxcb-present.so.0
      ln -sf $DEBLIB/libxcb-sync.so.1 /usr/lib64/libxcb-sync.so.1
      ln -sf $DEBLIB/libxcb-xfixes.so.0 /usr/lib64/libxcb-xfixes.so.0
      ln -sf $DEBLIB/libxcb-randr.so.0 /usr/lib64/libxcb-randr.so.0
      ln -sf $DEBLIB/libxcb-glx.so.0 /usr/lib64/libxcb-glx.so.0
      ln -sf $DEBLIB/libxshmfence.so.1 /usr/lib64/libxshmfence.so.1
      
      # Wayland
      ln -sf $DEBLIB/libwayland-client.so.0 /usr/lib64/libwayland-client.so.0
      ln -sf $DEBLIB/libwayland-server.so.0 /usr/lib64/libwayland-server.so.0
      
      # X11 auth
      ln -sf $DEBLIB/libXau.so.6 /usr/lib64/libXau.so.6
      ln -sf $DEBLIB/libXdmcp.so.6 /usr/lib64/libXdmcp.so.6
      ln -sf $DEBLIB/libbsd.so.0 /usr/lib64/libbsd.so.0
      ln -sf $DEBLIB/libmd.so.0 /usr/lib64/libmd.so.0
    ' || {
        echo "    Warning: Failed to create symlinks on $node"
      }
  done

  echo "Mesa installation complete"
}

function node::check_mesa() {
  # Check if mesa libraries are available
  if [ -f "/usr/lib64/libEGL.so.1" ] || [ -f "/usr/lib/x86_64-linux-gnu/libEGL.so.1" ]; then
    echo "Mesa libraries found on host"
    return 0
  else
    echo "Warning: Mesa libraries not found on host"
    echo "vGPU display may not work without OpenGL libraries"
    return 1
  fi
}

