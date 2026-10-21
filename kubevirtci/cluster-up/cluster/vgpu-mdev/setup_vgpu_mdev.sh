#!/bin/bash

set -xe

SCRIPT_PATH=$(dirname "$(realpath "$0")")
FAKE_VGPU_DIR="$SCRIPT_PATH/fake-nvidia-vgpu"

# Set KUBEVIRTCI_PATH if not set
if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(cd "$(dirname "$0")/../.." && pwd)"
fi
KUBEVIRTCI_PATH="$(cd "$KUBEVIRTCI_PATH" && pwd)/"

# Set KUBEVIRTCI_CONFIG_PATH if not set
if [ -z "$KUBEVIRTCI_CONFIG_PATH" ]; then
    KUBEVIRTCI_CONFIG_PATH="${HOME}/.kubevirtci"
fi

# Set KUBEVIRT_PROVIDER if not set
if [ -z "$KUBEVIRT_PROVIDER" ]; then
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

# Install build dependencies on a node
function install_build_deps() {
    local node_name=$1

    echo "===== Installing build dependencies on $node_name ====="

    ssh_to_node "$node_name" "sudo dnf install -y kernel-devel-\$(uname -r) gcc make elfutils-libelf-devel 2>&1 | tail -10"

    # Verify installation
    ssh_to_node "$node_name" "echo 'Verifying build environment...' && ls /lib/modules/\$(uname -r)/build/Makefile >/dev/null && echo '✓ kernel-devel OK' && which make gcc >/dev/null && echo '✓ build tools OK'"
}

# Copy kernel module source to node
function copy_module_source() {
    local node_name=$1

    echo "===== Copying kernel module source to $node_name ====="

    local tarball="/tmp/fake-nvidia-vgpu-$$.tar.gz"
    tar czf "$tarball" -C "$SCRIPT_PATH" \
        fake-nvidia-vgpu/Makefile \
        fake-nvidia-vgpu/fake-nvidia-vgpu.c \
        fake-nvidia-vgpu/compat.h \
        fake-nvidia-vgpu/dkms.conf \
        fake-nvidia-vgpu/setup-fake-vgpu.sh

    # Use the container + scp approach (more reliable for large files)
    local container_name="${KUBEVIRT_PROVIDER}-${node_name}"
    local container_runtime

    if [[ "${KUBEVIRTCI_RUNTIME:-}" == "podman" ]]; then
        container_runtime=podman
    elif [[ "${KUBEVIRTCI_RUNTIME:-}" == "docker" ]]; then
        container_runtime=docker
    elif command -v podman >/dev/null 2>&1 && podman ps >/dev/null 2>&1; then
        container_runtime=podman
    elif command -v docker >/dev/null 2>&1 && docker ps >/dev/null 2>&1; then
        container_runtime=docker
    else
        echo "ERROR: no usable Docker or Podman runtime found" >&2
        rm -f "$tarball"
        return 1
    fi

    # Copy tarball to container
    "$container_runtime" cp "$tarball" "${container_name}:/tmp/fake-nvidia-vgpu.tar.gz"

    # Get VM IP from node (node01=101, node02=102, etc.)
    local node_num="${node_name#node}"
    local vm_ip="192.168.66.1${node_num}"

    # Copy from container to VM
    "$container_runtime" exec "${container_name}" bash -c "
        scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i /vagrant.key \
            /tmp/fake-nvidia-vgpu.tar.gz root@${vm_ip}:/tmp/
    "

    # Extract on VM
    ssh_to_node "$node_name" "cd /tmp && sudo rm -rf fake-nvidia-vgpu && tar xzf fake-nvidia-vgpu.tar.gz && ls -la fake-nvidia-vgpu/"

    rm -f "$tarball"
}

# Compile kernel module on node
function compile_module() {
    local node_name=$1

    echo "===== Compiling kernel module on $node_name ====="

    ssh_to_node "$node_name" "cd /tmp/fake-nvidia-vgpu && make clean 2>/dev/null || true && make && ls -la *.ko"
}

# Load kernel module on node
function load_module() {
    local node_name=$1

    echo "===== Loading kernel module on $node_name ====="

    # Load dependencies (vfio and mdev modules) then our module
    ssh_to_node "$node_name" "echo 'Loading vfio and mdev dependencies...' && sudo modprobe vfio && sudo modprobe mdev && lsmod | grep -E '(vfio|mdev)' | head -5"

    # Load our module
    ssh_to_node "$node_name" "cd /tmp/fake-nvidia-vgpu && sudo rmmod fake-nvidia-vgpu 2>/dev/null || true"
    ssh_to_node "$node_name" "cd /tmp/fake-nvidia-vgpu && sudo insmod fake-nvidia-vgpu.ko && echo 'Module loaded:' && lsmod | grep fake_nvidia_vgpu"
    ssh_to_node "$node_name" "echo 'mdev_bus:' && ls -la /sys/class/mdev_bus/ 2>/dev/null || echo 'No mdev_bus yet'"
    ssh_to_node "$node_name" "echo 'Supported mdev types:' && ls /sys/class/mdev_bus/nvidia/mdev_supported_types/ 2>/dev/null || echo 'No types yet'"
}


# Set SELinux permissive mode on a node (dev-only workaround)
function set_selinux_permissive() {
    local node_name=$1

    echo "===== Setting SELinux permissive on $node_name ====="
    ssh_to_node "$node_name" "if command -v getenforce >/dev/null 2>&1; then
  mode=\$(getenforce)
  if [ \"\$mode\" = \"Enforcing\" ]; then
    sudo setenforce 0 || true
  fi
  sudo sed -ri 's/^SELINUX=.*/SELINUX=permissive/' /etc/selinux/config 2>/dev/null || true
  echo \"SELinux mode: \$(getenforce)\"
else
  echo 'SELinux tools not available'
fi"
}

function maybe_set_selinux_permissive() {
    local node_name=$1
    if [ "${ENABLE_SELINUX_PERMISSIVE:-false}" = "true" ]; then
        set_selinux_permissive "$node_name"
    else
        echo "===== Skipping SELinux permissive on $node_name (ENABLE_SELINUX_PERMISSIVE!=true) ====="
    fi
}

# Setup fake vGPU on a single node (full workflow)
function setup_node_vgpu() {
    local node_name=$1

    echo ""
    echo "########################################"
    echo "# Setting up fake vGPU on $node_name"
    echo "########################################"
    echo ""

    install_build_deps "$node_name"
    copy_module_source "$node_name"
    compile_module "$node_name"
    load_module "$node_name"
    # A/B result: SELinux alone was not the blocker; keep optional.
    # maybe_set_selinux_permissive "$node_name"

    # Label the node
    kubectl label node "$node_name" "$VGPU_NODE_LABEL" --overwrite

    echo ""
    echo "===== Fake vGPU setup completed on $node_name ====="
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
    echo "  full-setup      - Set up fake vGPU support on worker nodes (alias for setup)"
    echo ""
    echo "Environment flags:"
    echo "  ENABLE_SELINUX_PERMISSIVE=true   # optional dev workaround (default: false)"
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

# Resolve the worker list before entering the loop so kubectl failures cannot
# be hidden by command substitution in a for-loop word list.
function setup_all_worker_nodes() {
    local nodes
    nodes=$(get_worker_nodes) || {
        echo "ERROR: Could not query worker nodes; check that the cluster is running and KUBECONFIG points to it." >&2
        return 1
    }

    if [ -z "$nodes" ]; then
        echo "ERROR: No worker nodes found; refusing to report vGPU setup as successful." >&2
        return 1
    fi

    # Use a for-loop here because ssh/docker commands in setup_node_vgpu may
    # read stdin; that would otherwise consume the remaining lines from a
    # while-read loop and silently skip later workers.
    for node in $nodes; do
        setup_node_vgpu "$node"
    done
}

# Main
command=${1:-setup}
target_node=${2:-}

case "$command" in
    setup)
        if [ -n "$target_node" ]; then
            setup_node_vgpu "$target_node"
        else
            setup_all_worker_nodes
        fi
        ;;
    compile)
        if [ -n "$target_node" ]; then
            copy_module_source "$target_node"
            compile_module "$target_node"
        else
            for node in $(get_worker_nodes); do
                copy_module_source "$node"
                compile_module "$node"
            done
        fi
        ;;
    load)
        if [ -n "$target_node" ]; then
            load_module "$target_node"
        else
            for node in $(get_worker_nodes); do
                load_module "$node"
            done
        fi
        ;;
    unload)
        if [ -n "$target_node" ]; then
            unload_module "$target_node"
        else
            for node in $(get_worker_nodes); do
                unload_module "$node"
            done
        fi
        ;;
    status)
        if [ -n "$target_node" ]; then
            check_module_status "$target_node"
        else
            for node in $(get_worker_nodes); do
                check_module_status "$node"
            done
        fi
        ;;
    deps)
        if [ -n "$target_node" ]; then
            install_build_deps "$target_node"
        else
            for node in $(get_worker_nodes); do
                install_build_deps "$node"
            done
        fi
        ;;
    full-setup)
        setup_all_worker_nodes
        ;;
    help|-h|--help)
        usage
        ;;
    *)
        echo "Unknown command: $command"
        usage
        exit 1
        ;;
esac

echo ""
echo "Done!"
