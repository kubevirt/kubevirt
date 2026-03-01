#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
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
# Setup fake-sriov-vgpu module for testing NVIDIA SR-IOV vGPU VF discovery
#
# This module creates real PCI devices in /sys/bus/pci/devices/ by
# registering a virtual PCI bus.
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_PATH="${SCRIPT_DIR}/fake-sriov-vgpu.ko"
MODULE_NAME="fake_sriov_vgpu"

# Default configuration
NUM_VFS=${FAKE_SRIOV_VFS:-4}
VGPU_TYPE=${FAKE_SRIOV_VGPU_TYPE:-256}  # Non-zero means vGPU profile assigned

# Control interface path
CTRL_PATH="/sys/class/fake-sriov-vgpu/control"

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
        log_info "  cd ${SCRIPT_DIR} && make"
        exit 1
    fi
}

load_module() {
    log_info "Loading fake-sriov-vgpu module..."
    
    # Unload if already loaded
    if lsmod | grep -q "$MODULE_NAME"; then
        log_info "Module already loaded, removing first..."
        echo 1 > "$CTRL_PATH/clear" 2>/dev/null || true
        rmmod $MODULE_NAME 2>/dev/null || true
        sleep 1
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

create_vfs() {
    log_info "Creating $NUM_VFS fake VF devices..."
    
    if [[ ! -f "$CTRL_PATH/create" ]]; then
        log_error "Control interface not found at $CTRL_PATH"
        exit 1
    fi
    
    local created=0
    for ((i=0; i<NUM_VFS; i++)); do
        local slot=$((i % 32))
        local func=$((i / 32))
        
        # Create VF: slot func vgpu_type
        if echo "${slot} ${func} ${VGPU_TYPE}" > "$CTRL_PATH/create" 2>/dev/null; then
            log_info "Created VF: slot=$slot func=$func (vgpu_type=$VGPU_TYPE)"
            created=$((created + 1))
        else
            log_warn "Failed to create VF slot=$slot func=$func"
            dmesg | tail -5
        fi
    done
    
    log_info "Created $created VF devices"
    
    if [[ $created -eq 0 ]]; then
        log_error "Failed to create any VF devices"
        exit 1
    fi
}

show_status() {
    echo ""
    log_info "=== Fake SR-IOV vGPU Status ==="
    echo ""
    
    echo "Loaded modules:"
    lsmod | grep "$MODULE_NAME" || echo "  Module not loaded"
    echo ""
    
    echo "Created VFs:"
    if [[ -f "$CTRL_PATH/list" ]]; then
        cat "$CTRL_PATH/list"
    else
        echo "  Control interface not available"
    fi
    echo ""
    
    echo "PCI devices (fake domain 0001):"
    ls -la /sys/bus/pci/devices/ 2>/dev/null | grep "0001:" || echo "  None"
    echo ""
    
    # Show first VF details
    local first_vf=$(ls -d /sys/bus/pci/devices/0001:* 2>/dev/null | head -1)
    if [[ -n "$first_vf" && -d "$first_vf" ]]; then
        echo "Example VF: $first_vf"
        echo "  vendor: $(cat "$first_vf/vendor" 2>/dev/null)"
        echo "  device: $(cat "$first_vf/device" 2>/dev/null)"
        if [[ -f "$first_vf/nvidia/current_vgpu_type" ]]; then
            echo "  nvidia/current_vgpu_type: $(cat "$first_vf/nvidia/current_vgpu_type" 2>/dev/null)"
        fi
    fi
}

cleanup() {
    log_info "Cleaning up fake SR-IOV vGPU..."
    
    # Clear VFs
    if [[ -f "$CTRL_PATH/clear" ]]; then
        echo 1 > "$CTRL_PATH/clear" 2>/dev/null || true
    fi
    
    # Unload module
    if lsmod | grep -q "$MODULE_NAME"; then
        rmmod $MODULE_NAME 2>/dev/null || true
        log_info "Module unloaded"
    fi
}

usage() {
    echo "Usage: $0 [setup|cleanup|status|create-vf|remove-vf]"
    echo ""
    echo "Commands:"
    echo "  setup       Load module and create VFs (default)"
    echo "  cleanup     Clear VFs and unload module"
    echo "  status      Show current status"
    echo "  create-vf   Create a single VF: $0 create-vf SLOT FUNC [vgpu_type]"
    echo "  remove-vf   Remove a single VF: $0 remove-vf SLOT FUNC"
    echo ""
    echo "Environment variables:"
    echo "  FAKE_SRIOV_VFS         Number of VFs to create (default: 4)"
    echo "  FAKE_SRIOV_VGPU_TYPE   vGPU type value (default: 256, 0=no profile)"
    echo ""
    echo "Examples:"
    echo "  # Setup with defaults (4 VFs with vGPU type 256)"
    echo "  sudo $0 setup"
    echo ""
    echo "  # Setup with 8 VFs"
    echo "  sudo FAKE_SRIOV_VFS=8 $0 setup"
    echo ""
    echo "  # Create a VF without vGPU profile"
    echo "  sudo $0 create-vf 5 0 0"
    echo ""
    echo "  # Check the PCI devices"
    echo "  ls /sys/bus/pci/devices/0001:*"
}

# Main
case "${1:-setup}" in
    setup)
        check_root
        check_module_exists
        load_module
        create_vfs
        show_status
        ;;
    cleanup)
        check_root
        cleanup
        ;;
    status)
        show_status
        ;;
    create-vf)
        check_root
        if [[ -z "$2" || -z "$3" ]]; then
            log_error "Usage: $0 create-vf SLOT FUNC [vgpu_type]"
            exit 1
        fi
        slot="$2"
        func="$3"
        vgpu_type="${4:-256}"
        echo "${slot} ${func} ${vgpu_type}" > "$CTRL_PATH/create"
        log_info "Created VF: slot=$slot func=$func vgpu_type=$vgpu_type"
        ;;
    remove-vf)
        check_root
        if [[ -z "$2" || -z "$3" ]]; then
            log_error "Usage: $0 remove-vf SLOT FUNC"
            exit 1
        fi
        slot="$2"
        func="$3"
        echo "${slot} ${func}" > "$CTRL_PATH/remove"
        log_info "Removed VF: slot=$slot func=$func"
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
