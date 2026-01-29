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
# Setup fake-nvidia-vgpu module on the host for Kind-based vGPU testing
#
# This script:
# 1. Loads required VFIO modules
# 2. Loads the fake-nvidia-vgpu module
# 3. Creates mdev instances for testing
#
# Must be run as root before starting the Kind cluster.
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_PATH="${SCRIPT_DIR}/fake-nvidia-vgpu/fake-nvidia-vgpu.ko"
MODULE_NAME="fake_nvidia_vgpu"
# The mdev parent device will be named "nvidia" to match KubeVirt test expectations
MDEV_PARENT_NAME="nvidia"

# Number of mdev instances to create (default: 4)
NUM_INSTANCES=${FAKE_VGPU_INSTANCES:-4}

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_module_exists() {
    if [[ ! -f "$MODULE_PATH" ]]; then
        log_error "Module not found at $MODULE_PATH"
        log_info "Please build the module first:"
        log_info "  cd ${SCRIPT_DIR}/fake-nvidia-vgpu && make"
        exit 1
    fi
}

load_vfio_modules() {
    log_info "Loading VFIO modules..."
    
    modprobe vfio || true
    modprobe vfio_iommu_type1 || true
    modprobe mdev || true
    
    # Ensure /dev/vfio exists
    if [[ ! -d /dev/vfio ]]; then
        mkdir -p /dev/vfio
    fi
    
    if [[ ! -c /dev/vfio/vfio ]]; then
        mknod /dev/vfio/vfio c 10 196 2>/dev/null || true
    fi
    
    chmod 666 /dev/vfio/vfio 2>/dev/null || true
    
    log_info "VFIO modules loaded"
}

load_fake_vgpu_module() {
    log_info "Loading fake-nvidia-vgpu module..."
    
    # Unload if already loaded
    if lsmod | grep -q "$MODULE_NAME"; then
        log_info "Module already loaded, removing first..."
        # Remove any mdev instances first
        for dev in /sys/bus/mdev/devices/*; do
            if [[ -d "$dev" ]]; then
                echo 1 > "$dev/remove" 2>/dev/null || true
            fi
        done
        rmmod $MODULE_NAME 2>/dev/null || true
    fi
    
    # Load the module
    insmod "$MODULE_PATH"
    
    if lsmod | grep -q "$MODULE_NAME"; then
        log_info "Module loaded successfully"
    else
        log_error "Failed to load module"
        dmesg | tail -20
        exit 1
    fi
    
    # Wait for sysfs
    sleep 1
}

create_mdev_instances() {
    log_info "Creating $NUM_INSTANCES mdev instances..."
    
    local parent_device
    parent_device=$(ls /sys/class/mdev_bus/ 2>/dev/null | head -1)
    
    if [[ -z "$parent_device" ]]; then
        log_error "No mdev parent device found"
        exit 1
    fi
    
    log_info "Found mdev parent device: $parent_device"
    
    # Find the nvidia-222 type directory
    # The directory name is "nvidia-222" (driver_name + sysfs_name: "nvidia" + "222")
    local type_path="/sys/class/mdev_bus/$parent_device/mdev_supported_types/nvidia-222"
    
    if [[ ! -d "$type_path" ]]; then
        log_warn "nvidia-222 type not found at $type_path"
        log_info "Available types:"
        ls /sys/class/mdev_bus/$parent_device/mdev_supported_types/
        # Try to find any type with 222 in the name
        type_path=$(find /sys/class/mdev_bus/$parent_device/mdev_supported_types -maxdepth 1 -name "*222" | head -1)
        if [[ -z "$type_path" ]]; then
            log_error "No suitable mdev type found"
            exit 1
        fi
        log_info "Using type: $type_path"
    fi
    
    local create_path="${type_path}/create"
    local created=0
    
    for ((i=0; i<NUM_INSTANCES; i++)); do
        local uuid
        uuid=$(cat /proc/sys/kernel/random/uuid)
        
        if echo "$uuid" > "$create_path" 2>/dev/null; then
            log_info "Created mdev: $uuid"
            created=$((created + 1))
        else
            log_warn "Failed to create instance $((i+1))"
        fi
    done
    
    log_info "Created $created mdev instances"
    
    if [[ $created -eq 0 ]]; then
        log_error "Failed to create any mdev instances"
        exit 1
    fi
}

show_status() {
    echo ""
    log_info "=== Fake vGPU Status ==="
    echo ""
    
    echo "Loaded modules:"
    lsmod | grep -E "vfio|mdev|fake" || echo "  None"
    echo ""
    
    echo "mdev_bus:"
    ls -la /sys/class/mdev_bus/ 2>/dev/null || echo "  Not available"
    echo ""
    
    echo "Active mdev devices:"
    ls /sys/bus/mdev/devices/ 2>/dev/null || echo "  None"
    echo ""
    
    echo "VFIO devices:"
    ls -la /dev/vfio/ 2>/dev/null || echo "  Not available"
}

cleanup() {
    log_info "Cleaning up fake vGPU..."
    
    # Remove mdev instances
    for dev in /sys/bus/mdev/devices/*; do
        if [[ -d "$dev" ]]; then
            echo 1 > "$dev/remove" 2>/dev/null || true
        fi
    done
    
    # Unload module
    if lsmod | grep -q "$MODULE_NAME"; then
        rmmod $MODULE_NAME 2>/dev/null || true
        log_info "Module unloaded"
    fi
}

# Hotplug emulation functions
# These allow simulating device disappear/reappear for testing virt-handler hot-plug detection

HOTPLUG_CONTROL="/sys/class/nvidia/nvidia/hotplug_control"

hotplug_hide() {
    log_info "Hiding fake vGPU device (simulating device removal)..."
    
    if [[ ! -f "$HOTPLUG_CONTROL" ]]; then
        log_error "Hotplug control not found at $HOTPLUG_CONTROL"
        log_error "Make sure the fake-nvidia-vgpu module is loaded"
        exit 1
    fi
    
    # First remove all mdev instances
    for dev in /sys/bus/mdev/devices/*; do
        if [[ -d "$dev" ]]; then
            log_info "Removing mdev instance: $(basename $dev)"
            echo 1 > "$dev/remove" 2>/dev/null || true
        fi
    done
    
    echo "hide" > "$HOTPLUG_CONTROL"
    
    # Verify
    local state
    state=$(cat "$HOTPLUG_CONTROL")
    if [[ "$state" == "hidden" ]]; then
        log_info "Device is now hidden"
    else
        log_error "Failed to hide device, state: $state"
        exit 1
    fi
}

hotplug_show() {
    log_info "Showing fake vGPU device (simulating device insertion)..."
    
    if [[ ! -f "$HOTPLUG_CONTROL" ]]; then
        log_error "Hotplug control not found at $HOTPLUG_CONTROL"
        log_error "Make sure the fake-nvidia-vgpu module is loaded"
        exit 1
    fi
    
    echo "show" > "$HOTPLUG_CONTROL"
    
    # Verify
    local state
    state=$(cat "$HOTPLUG_CONTROL")
    if [[ "$state" == "visible" ]]; then
        log_info "Device is now visible"
    else
        log_error "Failed to show device, state: $state"
        exit 1
    fi
}

hotplug_status() {
    if [[ ! -f "$HOTPLUG_CONTROL" ]]; then
        echo "Hotplug control not available (module not loaded?)"
        return 1
    fi
    
    local state
    state=$(cat "$HOTPLUG_CONTROL")
    echo "Hotplug state: $state"
    echo "Control file: $HOTPLUG_CONTROL"
}

usage() {
    echo "Usage: $0 [setup|cleanup|status|hide|show]"
    echo ""
    echo "Commands:"
    echo "  setup    Load module and create mdev instances (default)"
    echo "  cleanup  Remove mdev instances and unload module"
    echo "  status   Show current status"
    echo "  hide     Hide device (simulate hot-unplug for testing)"
    echo "  show     Show device (simulate hot-plug for testing)"
    echo ""
    echo "Environment variables:"
    echo "  FAKE_VGPU_INSTANCES  Number of mdev instances to create (default: 4)"
    echo ""
    echo "Hotplug emulation:"
    echo "  The hide/show commands allow testing virt-handler's hot-plug detection."
    echo "  Use 'hide' to simulate device removal, 'show' to simulate device insertion."
}

# Main
case "${1:-setup}" in
    setup)
        check_root
        check_module_exists
        load_vfio_modules
        load_fake_vgpu_module
        create_mdev_instances
        show_status
        ;;
    cleanup)
        check_root
        cleanup
        ;;
    status)
        show_status
        hotplug_status
        ;;
    hide)
        check_root
        hotplug_hide
        ;;
    show)
        check_root
        hotplug_show
        ;;
    -h|--help)
        usage
        ;;
    *)
        log_error "Unknown command: $1"
        usage
        exit 1
        ;;
esac
