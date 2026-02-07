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
#
# Setup script for fake NVIDIA vGPU module
#
# This script:
# 1. Builds the fake-nvidia-vgpu kernel module
# 2. Loads it into the kernel
# 3. Creates the required mdev instances
# 4. Sets up VFIO infrastructure
#
# Usage:
#   ./setup-fake-vgpu.sh [--instances N]
#
# Options:
#   --instances N    Number of mdev instances to create (default: 4)
#   --type TYPE      mdev type: nvidia-222 (T4-1B) or nvidia-223 (T4-2B) (default: nvidia-222)
#   --clean          Clean build artifacts before building
#   --unload         Unload module and remove mdev instances
#   --status         Show current status
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_NAME="fake-nvidia-vgpu"
DEFAULT_INSTANCES=4
DEFAULT_TYPE="nvidia-222"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_dependencies() {
    local missing=()
    
    if ! command -v make &>/dev/null; then
        missing+=("make")
    fi
    
    if ! command -v gcc &>/dev/null; then
        missing+=("gcc")
    fi
    
    if [[ ! -d "/lib/modules/$(uname -r)/build" ]]; then
        missing+=("kernel-headers")
    fi
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing dependencies: ${missing[*]}"
        log_info "Install with:"
        log_info "  Fedora/RHEL: dnf install kernel-devel kernel-headers gcc make"
        log_info "  Ubuntu/Debian: apt install linux-headers-\$(uname -r) build-essential"
        exit 1
    fi
}

setup_vfio() {
    log_info "Setting up VFIO infrastructure..."
    
    # Load VFIO modules
    modprobe vfio 2>/dev/null || true
    modprobe vfio_iommu_type1 2>/dev/null || true
    modprobe vfio-pci 2>/dev/null || true
    
    # Create /dev/vfio if it doesn't exist
    if [[ ! -d /dev/vfio ]]; then
        mkdir -p /dev/vfio
    fi
    
    # Create /dev/vfio/vfio if it doesn't exist
    if [[ ! -c /dev/vfio/vfio ]]; then
        mknod /dev/vfio/vfio c 10 196 2>/dev/null || true
    fi
    
    chmod 666 /dev/vfio/vfio 2>/dev/null || true
    
    log_info "VFIO setup complete"
}

build_module() {
    log_info "Building ${MODULE_NAME} module..."
    
    cd "$SCRIPT_DIR"
    
    if [[ "$1" == "--clean" ]]; then
        make clean
    fi
    
    make
    
    if [[ ! -f "${MODULE_NAME}.ko" ]]; then
        log_error "Build failed - ${MODULE_NAME}.ko not found"
        exit 1
    fi
    
    log_info "Build successful"
}

load_module() {
    log_info "Loading ${MODULE_NAME} module..."
    
    # Unload if already loaded
    rmmod ${MODULE_NAME} 2>/dev/null || true
    
    # Load the module
    insmod "${SCRIPT_DIR}/${MODULE_NAME}.ko"
    
    # Verify it loaded
    if lsmod | grep -q "${MODULE_NAME//-/_}"; then
        log_info "Module loaded successfully"
    else
        log_error "Failed to load module"
        dmesg | tail -20
        exit 1
    fi
    
    # Wait for sysfs to be ready
    sleep 1
    
    # Verify mdev_bus is available
    if [[ -d "/sys/class/mdev_bus/${MODULE_NAME//-/_}" ]]; then
        log_info "mdev_bus available at /sys/class/mdev_bus/${MODULE_NAME//-/_}"
    else
        log_warn "mdev_bus not found - checking alternative paths..."
        ls -la /sys/class/mdev_bus/ 2>/dev/null || log_error "No mdev_bus found"
    fi
}

create_mdev_instances() {
    local num_instances=$1
    local mdev_type=$2
    local parent_device
    
    log_info "Creating ${num_instances} mdev instances of type ${mdev_type}..."
    
    # Find the parent device
    parent_device=$(ls /sys/class/mdev_bus/ 2>/dev/null | head -1)
    
    if [[ -z "$parent_device" ]]; then
        log_error "No mdev parent device found"
        exit 1
    fi
    
    local create_path="/sys/class/mdev_bus/${parent_device}/mdev_supported_types/${mdev_type}/create"
    
    if [[ ! -f "$create_path" ]]; then
        log_error "mdev type ${mdev_type} not found at ${create_path}"
        log_info "Available types:"
        ls "/sys/class/mdev_bus/${parent_device}/mdev_supported_types/" 2>/dev/null || echo "None"
        exit 1
    fi
    
    # Check available instances
    local available
    available=$(cat "/sys/class/mdev_bus/${parent_device}/mdev_supported_types/${mdev_type}/available_instances")
    
    if [[ $num_instances -gt $available ]]; then
        log_warn "Requested ${num_instances} instances but only ${available} available"
        num_instances=$available
    fi
    
    # Create instances
    local created=0
    for ((i=0; i<num_instances; i++)); do
        local uuid
        uuid=$(cat /proc/sys/kernel/random/uuid)
        
        if echo "$uuid" > "$create_path" 2>/dev/null; then
            log_info "Created mdev instance: $uuid"
            ((created++))
        else
            log_warn "Failed to create instance $((i+1))"
        fi
    done
    
    log_info "Created ${created}/${num_instances} mdev instances"
    
    # Show created devices
    log_info "Active mdev devices:"
    ls /sys/bus/mdev/devices/ 2>/dev/null || echo "None"
}

remove_mdev_instances() {
    log_info "Removing all mdev instances..."
    
    local count=0
    for device in /sys/bus/mdev/devices/*; do
        if [[ -d "$device" ]]; then
            local uuid
            uuid=$(basename "$device")
            if echo 1 > "${device}/remove" 2>/dev/null; then
                log_info "Removed: $uuid"
                ((count++))
            fi
        fi
    done
    
    log_info "Removed ${count} mdev instances"
}

unload_module() {
    log_info "Unloading ${MODULE_NAME} module..."
    
    # Remove mdev instances first
    remove_mdev_instances
    
    # Unload module
    if lsmod | grep -q "${MODULE_NAME//-/_}"; then
        rmmod ${MODULE_NAME}
        log_info "Module unloaded"
    else
        log_info "Module was not loaded"
    fi
}

show_status() {
    echo "=== Fake NVIDIA vGPU Status ==="
    echo ""
    
    echo "Module status:"
    if lsmod | grep -q "${MODULE_NAME//-/_}"; then
        lsmod | grep "${MODULE_NAME//-/_}"
    else
        echo "  Not loaded"
    fi
    echo ""
    
    echo "mdev_bus:"
    if [[ -d /sys/class/mdev_bus ]]; then
        ls -la /sys/class/mdev_bus/
    else
        echo "  Not available"
    fi
    echo ""
    
    echo "Supported types:"
    for parent in /sys/class/mdev_bus/*; do
        if [[ -d "$parent/mdev_supported_types" ]]; then
            for type_dir in "$parent/mdev_supported_types"/*; do
                if [[ -d "$type_dir" ]]; then
                    local type_name
                    type_name=$(basename "$type_dir")
                    local pretty_name available
                    pretty_name=$(cat "$type_dir/name" 2>/dev/null || echo "N/A")
                    available=$(cat "$type_dir/available_instances" 2>/dev/null || echo "N/A")
                    echo "  ${type_name}: ${pretty_name} (${available} available)"
                fi
            done
        fi
    done
    echo ""
    
    echo "Active mdev devices:"
    if [[ -d /sys/bus/mdev/devices ]] && [[ -n "$(ls -A /sys/bus/mdev/devices 2>/dev/null)" ]]; then
        for device in /sys/bus/mdev/devices/*; do
            local uuid mdev_type
            uuid=$(basename "$device")
            mdev_type=$(basename "$(readlink -f "$device/mdev_type")" 2>/dev/null || echo "unknown")
            echo "  ${uuid} (type: ${mdev_type})"
        done
    else
        echo "  None"
    fi
    echo ""
    
    echo "VFIO devices:"
    ls -la /dev/vfio/ 2>/dev/null || echo "  Not available"
}

usage() {
    echo "Usage: $0 [OPTIONS] COMMAND"
    echo ""
    echo "Commands:"
    echo "  setup       Build, load module, and create mdev instances"
    echo "  build       Build the kernel module only"
    echo "  load        Load the kernel module only"
    echo "  create      Create mdev instances only"
    echo "  unload      Remove mdev instances and unload module"
    echo "  status      Show current status"
    echo ""
    echo "Options:"
    echo "  --instances N    Number of mdev instances to create (default: ${DEFAULT_INSTANCES})"
    echo "  --type TYPE      mdev type: nvidia-222 or nvidia-223 (default: ${DEFAULT_TYPE})"
    echo "  --clean          Clean before building"
    echo "  -h, --help       Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 setup                    # Full setup with defaults"
    echo "  $0 setup --instances 8      # Create 8 mdev instances"
    echo "  $0 status                   # Show current status"
    echo "  $0 unload                   # Clean up everything"
}

# Parse arguments
INSTANCES=$DEFAULT_INSTANCES
MDEV_TYPE=$DEFAULT_TYPE
CLEAN=""
COMMAND=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --instances)
            INSTANCES="$2"
            shift 2
            ;;
        --type)
            MDEV_TYPE="$2"
            shift 2
            ;;
        --clean)
            CLEAN="--clean"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        setup|build|load|create|unload|status)
            COMMAND="$1"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

if [[ -z "$COMMAND" ]]; then
    usage
    exit 1
fi

# Execute command
case $COMMAND in
    setup)
        check_root
        check_dependencies
        setup_vfio
        build_module $CLEAN
        load_module
        create_mdev_instances "$INSTANCES" "$MDEV_TYPE"
        show_status
        ;;
    build)
        check_dependencies
        build_module $CLEAN
        ;;
    load)
        check_root
        setup_vfio
        load_module
        ;;
    create)
        check_root
        create_mdev_instances "$INSTANCES" "$MDEV_TYPE"
        ;;
    unload)
        check_root
        unload_module
        ;;
    status)
        show_status
        ;;
esac
