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
# Setup fake-iommu + fake-pci modules on the current Linux kernel for
# DRA PCI passthrough testing.
#
# What this does:
#   1. (optional, default ON) Loads fake-iommu - a no-op IOMMU that claims
#      devices on the synthetic PCI domain. Required for vfio-pci binding
#      and therefore required for "VMI reaches Running" tests.
#   2. Loads fake-pci - the synthetic PCI host bridge with N fake
#      PCI devices.
#
# Order matters: fake-iommu MUST be loaded first; fake-pci MUST be
# unloaded first. This script enforces both.
#
# Must be run as root.

set -e
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# fake-pci
PCI_MODULE_DIR="${SCRIPT_DIR}/fake-pci"
PCI_MODULE_PATH="${PCI_MODULE_DIR}/fake-pci.ko"
PCI_MODULE_NAME="fake_pci"

# fake-iommu
IOMMU_MODULE_DIR="${SCRIPT_DIR}/fake-iommu"
IOMMU_MODULE_PATH="${IOMMU_MODULE_DIR}/fake-iommu.ko"
IOMMU_MODULE_NAME="fake_iommu"

# Defaults / overrides
NUM_DEVICES="${FAKE_PCI_DEVICES:-8}"
PCI_DOMAIN_PARAM="${FAKE_PCI_DOMAIN:-0xfaca}"
VENDOR_ID_PARAM="${FAKE_PCI_VENDOR_ID:-}"
DEVICE_ID_PARAM="${FAKE_PCI_DEVICE_ID:-}"
WITH_IOMMU="${FAKE_IOMMU:-true}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

fatal() {
    echo "FATAL: $*" >&2
    exit 1
}

validate_root() {
    [ "$(id -u)" -eq 0 ] || fatal "This script must be run as root"
}

check_modules_exist() {
    if [[ ! -f "${PCI_MODULE_PATH}" ]]; then
        log_error "Module not found at ${PCI_MODULE_PATH}"
        log_info "Build it first:"
        log_info "  cd ${PCI_MODULE_DIR} && make"
        exit 1
    fi
    if [[ "${WITH_IOMMU}" == "true" ]] && [[ ! -f "${IOMMU_MODULE_PATH}" ]]; then
        log_error "Module not found at ${IOMMU_MODULE_PATH}"
        log_info "Build it first:"
        log_info "  cd ${IOMMU_MODULE_DIR} && make"
        log_info "Or set FAKE_IOMMU=false to skip it (discovery-only mode)."
        exit 1
    fi
}

is_loaded() {
    lsmod | awk -v m="$1" '$1 == m { found = 1 } END { exit !found }'
}

load_iommu() {
    if [[ "${WITH_IOMMU}" != "true" ]]; then
        log_info "Skipping fake-iommu (FAKE_IOMMU=${WITH_IOMMU})"
        return
    fi

    log_info "Loading ${IOMMU_MODULE_NAME} module (must be loaded BEFORE fake-pci)..."

    if is_loaded "${IOMMU_MODULE_NAME}"; then
        log_info "${IOMMU_MODULE_NAME} already loaded; reloading"
        # If fake-pci references fake-iommu, it must be unloaded first.
        if is_loaded "${PCI_MODULE_NAME}"; then
            rmmod "${PCI_MODULE_NAME}" 2>/dev/null || true
        fi
        rmmod "${IOMMU_MODULE_NAME}" 2>/dev/null || true
    fi

    local args=("target_domain=${PCI_DOMAIN_PARAM}")
    insmod "${IOMMU_MODULE_PATH}" "${args[@]}"

    if is_loaded "${IOMMU_MODULE_NAME}"; then
        log_info "${IOMMU_MODULE_NAME} loaded with args: ${args[*]}"
    else
        log_error "Failed to load ${IOMMU_MODULE_NAME}"
        dmesg | tail -30
        exit 1
    fi

    sleep 1

    if [[ ! -d /sys/class/iommu/fake-iommu ]]; then
        log_warn "/sys/class/iommu/fake-iommu not found - module may not have registered properly"
    fi
}

load_pci() {
    log_info "Loading ${PCI_MODULE_NAME} module..."

    if is_loaded "${PCI_MODULE_NAME}"; then
        log_info "${PCI_MODULE_NAME} already loaded; reloading"
        rmmod "${PCI_MODULE_NAME}" 2>/dev/null || true
    fi

    local args=("num_devices=${NUM_DEVICES}" "pci_domain=${PCI_DOMAIN_PARAM}")
    [[ -n "${VENDOR_ID_PARAM}" ]] && args+=("vendor_id=${VENDOR_ID_PARAM}")
    [[ -n "${DEVICE_ID_PARAM}" ]] && args+=("device_id=${DEVICE_ID_PARAM}")

    insmod "${PCI_MODULE_PATH}" "${args[@]}"

    if is_loaded "${PCI_MODULE_NAME}"; then
        log_info "${PCI_MODULE_NAME} loaded with args: ${args[*]}"
    else
        log_error "Failed to load ${PCI_MODULE_NAME}"
        dmesg | tail -30
        exit 1
    fi

    sleep 1
}

unload_pci() {
    if is_loaded "${PCI_MODULE_NAME}"; then
        log_info "Unloading ${PCI_MODULE_NAME}..."
        rmmod "${PCI_MODULE_NAME}"
    else
        log_info "${PCI_MODULE_NAME} not loaded"
    fi
}

unload_iommu() {
    if [[ "${WITH_IOMMU}" != "true" ]]; then
        return
    fi
    if is_loaded "${IOMMU_MODULE_NAME}"; then
        log_info "Unloading ${IOMMU_MODULE_NAME}..."
        rmmod "${IOMMU_MODULE_NAME}"
    else
        log_info "${IOMMU_MODULE_NAME} not loaded"
    fi
}

show_status() {
    echo ""
    log_info "=== Fake PCI + IOMMU Status ==="
    echo ""

    echo "Modules loaded:"
    lsmod | grep -E "${IOMMU_MODULE_NAME}|${PCI_MODULE_NAME}" || echo "  none"
    echo ""

    echo "Registered IOMMUs (/sys/class/iommu/):"
    if [[ -d /sys/class/iommu ]] && [[ -n "$(ls -A /sys/class/iommu 2>/dev/null)" ]]; then
        ls /sys/class/iommu/ | sed 's/^/  /'
    else
        echo "  (none)"
    fi
    echo ""

    echo "Fake PCI devices:"
    local found=0
    for dev in /sys/bus/pci/devices/*; do
        local bdf
        bdf=$(basename "${dev}")
        # Filter to non-domain-0000 entries (our private domain).
        if [[ "${bdf}" == 0000:* ]]; then
            continue
        fi
        local vendor device class iommu_group driver
        vendor=$(cat "${dev}/vendor" 2>/dev/null || echo "?")
        device=$(cat "${dev}/device" 2>/dev/null || echo "?")
        class=$(cat "${dev}/class" 2>/dev/null || echo "?")
        iommu_group=$(basename "$(readlink "${dev}/iommu_group" 2>/dev/null)" 2>/dev/null)
        driver=$(basename "$(readlink "${dev}/driver" 2>/dev/null)" 2>/dev/null)
        [[ -z "${iommu_group}" ]] && iommu_group="-"
        [[ -z "${driver}" ]] && driver="-"
        echo "  ${bdf}  vendor=${vendor} device=${device} class=${class} iommu_group=${iommu_group} driver=${driver}"
        found=$((found + 1))
    done
    if [[ ${found} -eq 0 ]]; then
        echo "  (none - fake-pci not loaded?)"
    fi
    echo ""

    echo "VFIO state:"
    ls /dev/vfio/ 2>/dev/null | sed 's/^/  /' || echo "  /dev/vfio not present"
}

# Bind/unbind helpers for testing vfio-pci on the fake devices

bind_vfio() {
    if [[ "${WITH_IOMMU}" != "true" ]]; then
        log_error "vfio-pci bind requires the fake-iommu module (FAKE_IOMMU=true)"
        exit 1
    fi
    local vendor="${FAKE_PCI_VENDOR_ID:-0xe1a5}"
    local device="${FAKE_PCI_DEVICE_ID:-0xd0c5}"
    log_info "Loading vfio-pci and binding vendor=${vendor} device=${device}..."
    modprobe vfio-pci 2>/dev/null || true
    # new_id wants raw hex without 0x prefix
    local vraw="${vendor#0x}"
    local draw="${device#0x}"
    echo "${vraw} ${draw}" > /sys/bus/pci/drivers/vfio-pci/new_id 2>/dev/null || true
    sleep 1
    log_info "Binding result:"
    for dev in /sys/bus/pci/devices/*; do
        local bdf
        bdf=$(basename "${dev}")
        [[ "${bdf}" == 0000:* ]] && continue
        local drv
        drv=$(basename "$(readlink "${dev}/driver" 2>/dev/null)" 2>/dev/null)
        echo "  ${bdf}  driver=${drv:-none}"
    done
    echo ""
    echo "/dev/vfio entries:"
    ls /dev/vfio/ 2>/dev/null | sed 's/^/  /' || echo "  (none)"
}

usage() {
    cat <<EOF
Usage: $0 [setup|cleanup|status|bind-vfio]

Commands:
  setup       Load fake-iommu (if enabled) THEN fake-pci. Default.
  cleanup     Unload fake-pci THEN fake-iommu.
  status      Show modules, devices, IOMMU groups, drivers, /dev/vfio.
  bind-vfio   Bind vfio-pci to the fake devices (requires FAKE_IOMMU=true).

Environment variables:
  FAKE_IOMMU            true|false (default: true)
                        Load the fake-iommu companion. Required for vfio-pci
                        binding and "VMI reaches Running" tests. Set false
                        for discovery-only testing.
  FAKE_PCI_DEVICES      Number of fake PCI devices to expose (default: 8)
  FAKE_PCI_DOMAIN       PCI domain number for the synthetic bridge
                        (default: 0xfaca; passed to BOTH modules to keep
                        their target domain in sync)
  FAKE_PCI_VENDOR_ID    PCI vendor ID to emulate (default: 0xe1a5)
  FAKE_PCI_DEVICE_ID    PCI device ID to emulate (default: 0xd0c5)

Examples:
  # Default: 4 fake PCI devices + fake-iommu, ready for VMI passthrough tests
  sudo $0 setup

  # Discovery-only (no IOMMU, no vfio-pci binding)
  FAKE_IOMMU=false sudo $0 setup

  # 8 fake PCI devices with a different synthetic device ID and domain
  FAKE_PCI_DEVICES=8 FAKE_PCI_DEVICE_ID=0xf00d FAKE_PCI_DOMAIN=0xfada \\
    sudo $0 setup

  # Inspect everything
  sudo $0 status
  lspci -D -nn -d e1a5:

  # Bind vfio-pci and verify /dev/vfio entries
  sudo $0 bind-vfio
EOF
}

case "${1:-setup}" in
    setup)
        validate_root
        check_modules_exist
        load_iommu
        load_pci
        show_status
        ;;
    cleanup)
        validate_root
        unload_pci
        unload_iommu
        ;;
    status)
        show_status
        ;;
    bind-vfio)
        validate_root
        bind_vfio
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
