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

set -euo pipefail

SCRIPT_PATH=$(dirname "$(realpath "$0")")

# Set KUBEVIRTCI_PATH if not set
if [ -z "${KUBEVIRTCI_PATH:-}" ]; then
    KUBEVIRTCI_PATH="$(cd "$(dirname "$0")/../.." && pwd)"
fi
KUBEVIRTCI_PATH="$(cd "$KUBEVIRTCI_PATH" && pwd)/"

# Set KUBEVIRT_PROVIDER if not set
if [ -z "${KUBEVIRT_PROVIDER:-}" ]; then
    export KUBEVIRT_PROVIDER="k8s-1.35"
fi

# Configuration
VGPU_NODE_LABEL_KEY="vgpu_capable"
VGPU_NODE_LABEL_VALUE="true"
VGPU_NODE_LABEL="$VGPU_NODE_LABEL_KEY=$VGPU_NODE_LABEL_VALUE"

# SSH function to execute commands on nodes
function ssh_to_node() {
    local node_name=$1
    shift
    "${KUBEVIRTCI_PATH}/ssh.sh" "$node_name" "$@"
}

function run_step() {
    local label=$1
    shift
    local log
    log=$(mktemp)

    printf '  %-30s' "$label"
    if "$@" >"$log" 2>&1; then
        echo 'OK'
        rm -f "$log"
    else
        echo 'FAILED' >&2
        cat "$log" >&2
        rm -f "$log"
        return 1
    fi
}

# Install build dependencies on a node
function install_build_deps() {
    local node_name=$1

    ssh_to_node "$node_name" "sudo dnf install -y kernel-devel-\$(uname -r) gcc make elfutils-libelf-devel" || return

    ssh_to_node "$node_name" "test -f /lib/modules/\$(uname -r)/build/Makefile && command -v make && command -v gcc" || return
}

# Copy kernel module source to node
function copy_module_source() {
    local node_name=$1
    local encoded_source

    encoded_source=$(tar -C "$SCRIPT_PATH" -cz \
        fake-nvidia-vgpu/Makefile \
        fake-nvidia-vgpu/fake-nvidia-vgpu.c \
        fake-nvidia-vgpu/compat.h | base64 | tr -d '\n') || return
    ssh_to_node "$node_name" "cd /tmp && sudo rm -rf fake-nvidia-vgpu && printf '%s' '$encoded_source' | base64 -d | tar xzf -"
}

# Compile kernel module on node
function compile_module() {
    local node_name=$1

    ssh_to_node "$node_name" "cd /tmp/fake-nvidia-vgpu && (make clean 2>/dev/null || true) && make && test -s fake-nvidia-vgpu.ko"
}

# Load kernel module on node
function load_module() {
    local node_name=$1

    # Load dependencies (vfio and mdev modules) then our module
    ssh_to_node "$node_name" "sudo modprobe vfio && sudo modprobe mdev" || return

    # Load our module
    ssh_to_node "$node_name" "if test -d /sys/module/fake_nvidia_vgpu; then sudo rmmod fake-nvidia-vgpu; fi" || return
    ssh_to_node "$node_name" "cd /tmp/fake-nvidia-vgpu && sudo insmod fake-nvidia-vgpu.ko"
}

function verify_module() {
    local node_name=$1
    local types=/sys/class/mdev_bus/nvidia/mdev_supported_types
    local default_profile="$types/nvidia-222"
    local override_profile="$types/nvidia-223"
    local check

    printf -v check "lsmod | grep -q '^fake_nvidia_vgpu ' && test -d %q && test \"\$(cat %q)\" = %q && grep -Eq %q %q && test \"\$(cat %q)\" = %q && test -d %q && test \"\$(cat %q)\" = %q && grep -Eq %q %q && test \"\$(cat %q)\" = %q && test \"\$(find %q -mindepth 1 -maxdepth 1 -type d | wc -l)\" = 2" \
        "$default_profile" "$default_profile/name" 'GRID T4-1B' \
        'max_instance=16$' "$default_profile/description" \
        "$default_profile/available_instances" 16 \
        "$override_profile" "$override_profile/name" 'GRID T4-2B' \
        'max_instance=8$' "$override_profile/description" \
        "$override_profile/available_instances" 8 "$types"

    ssh_to_node "$node_name" "$check" || {
        echo "Expected loaded module with nvidia-222 (16 instances) and nvidia-223 (8 instances)." >&2
        return 1
    }
}

function label_node() {
    local node_name=$1

    kubectl label node "$node_name" "$VGPU_NODE_LABEL" --overwrite >/dev/null || return
    [ "$(kubectl get node "$node_name" -o "jsonpath={.metadata.labels.$VGPU_NODE_LABEL_KEY}")" = "$VGPU_NODE_LABEL_VALUE" ]
}


# Setup fake vGPU on a single node (full workflow)
function setup_node_vgpu() {
    local node_name=$1

    echo "$node_name:"
    run_step 'Install build dependencies' install_build_deps "$node_name"
    run_step 'Copy module source' copy_module_source "$node_name"
    run_step 'Build kernel module' compile_module "$node_name"
    run_step 'Load kernel module' load_module "$node_name"
    run_step 'Verify module and mdev types' verify_module "$node_name"
    run_step 'Label worker node' label_node "$node_name"
}

# Check module status on a node
function check_module_status() {
    local node_name=$1

    echo "===== Module status on $node_name ====="
    ssh_to_node "$node_name" "echo '=== Loaded module ===' && (lsmod | grep fake_nvidia_vgpu || echo 'Module not loaded')"
    ssh_to_node "$node_name" "echo '=== mdev_bus ===' && (ls -la /sys/class/mdev_bus/ 2>/dev/null || echo 'No mdev_bus')"
    ssh_to_node "$node_name" "echo '=== Supported mdev types ===' && (find /sys/class/mdev_bus/*/mdev_supported_types -maxdepth 1 -type d 2>/dev/null | head -10 || echo 'No mdev types')"
    ssh_to_node "$node_name" "echo '=== dmesg (last 10 lines from module) ===' && (dmesg | grep -i 'fake.*vgpu' | tail -10 || echo 'No relevant dmesg')"
}

# Unload module from a node
function unload_module() {
    local node_name=$1

    echo "===== Unloading kernel module from $node_name ====="
    ssh_to_node "$node_name" "sudo rmmod fake-nvidia-vgpu 2>/dev/null || echo 'Module not loaded'"
}

# Print usage
function usage() {
    echo "Usage: $0 [command] [node]"
    echo ""
    echo "Commands:"
    echo "  setup [node]    - Full setup on specified node or all worker nodes"
    echo "  compile [node]  - Just compile module on node (deps must be installed)"
    echo "  load [node]     - Load compiled module on node"
    echo "  unload [node]   - Unload module from node"
    echo "  status [node]   - Check module status on node"
    echo "  deps [node]     - Install build dependencies on node"
    echo "  full-setup [node] - Alias for setup"
    echo ""
    echo "If no node is specified, operates on all worker nodes."
    echo ""
    echo "Examples:"
    echo "  $0 setup           # Setup on all workers"
    echo "  $0 setup node01    # Setup on node01 only"
    echo "  $0 status node02   # Check status on node02"
}

# Get worker nodes
function get_worker_nodes() {
    kubectl get nodes -l node-role.kubernetes.io/worker -o custom-columns=:.metadata.name --no-headers
}

function dispatch_node_command() {
    local command=$1
    local node=$2

    case "$command" in
        setup) setup_node_vgpu "$node" ;;
        compile)
            copy_module_source "$node"
            compile_module "$node"
            ;;
        load) load_module "$node" ;;
        unload) unload_module "$node" ;;
        status) check_module_status "$node" ;;
        deps) install_build_deps "$node" ;;
    esac
}

function dispatch_workers() {
    local command=$1
    local target_node=$2
    local nodes node

    if [ -n "$target_node" ]; then
        nodes=$target_node
    else
        nodes=$(get_worker_nodes) || {
            echo "ERROR: Could not query workers; check the cluster and KUBECONFIG." >&2
            return 1
        }
        if [ -z "$nodes" ]; then
            echo "ERROR: No worker nodes found." >&2
            return 1
        fi
    fi

    # SSH and container commands may read stdin, so do not use while read.
    for node in $nodes; do
        dispatch_node_command "$command" "$node"
    done
}

command=${1:-setup}
target_node=${2:-}

if (( $# > 2 )); then
    usage
    exit 1
fi

case "$command" in
    full-setup) command=setup ;;
    setup|compile|load|unload|status|deps) ;;
    help|-h|--help) usage; exit 0 ;;
    *)
        echo "Unknown command: $command" >&2
        usage
        exit 1
        ;;
esac

dispatch_workers "$command" "$target_node"
