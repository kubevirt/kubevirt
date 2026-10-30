#!/usr/bin/env bash
set -euo pipefail

source_dir=/tmp/vgpu-sriov

install_deps() {
    if ! test -f "/lib/modules/$(uname -r)/build/Makefile" ||
       ! command -v gcc >/dev/null || ! command -v make >/dev/null; then
        sudo dnf install -y "kernel-devel-$(uname -r)" gcc make elfutils-libelf-devel
    fi
}

build_modules() {
    local module
    for module in fake-iommu fake-pci fake-nvidia-vgpu; do
        sudo make -C "$source_dir/$module" clean >/dev/null 2>&1 || true
        sudo make -C "$source_dir/$module"
    done
}

load_modules() {
    local module
    sudo modprobe vfio-pci-core
    for module in fake-nvidia-sriov fake-pci fake-iommu; do
        if test -d "/sys/module/${module//-/_}"; then
            sudo rmmod "$module"
        fi
    done
    sudo insmod "$source_dir/fake-iommu/fake-iommu.ko"
    sudo insmod "$source_dir/fake-pci/fake-pci.ko"
    sudo insmod "$source_dir/fake-nvidia-vgpu/fake-nvidia-sriov.ko"
}

verify_devices() {
    local fn vf
    test "$(cat /sys/bus/pci/devices/faca:00:00.0/sriov_numvfs)" = 4
    for fn in 1 2 3 4; do
        vf="/sys/bus/pci/devices/faca:00:00.$fn"
        test -d "$vf"
        test -L "$vf/physfn"
        test "$(cat "$vf/vendor")" = 0x10de
        test "$(cat "$vf/device")" = 0x26b9
        test "$(basename "$(readlink "$vf/driver")")" = nvidia
        echo 1155 | sudo tee "$vf/nvidia/current_vgpu_type" >/dev/null
        test "$(cat "$vf/nvidia/current_vgpu_type")" = 1155
        test -d "$vf/vfio-dev"
        test -L "$vf/iommu_group"
    done
}

case "${1:-setup}" in
    deps) install_deps ;;
    build) build_modules ;;
    load) load_modules ;;
    verify) verify_devices ;;
    setup)
        install_deps
        build_modules
        load_modules
        verify_devices
        ;;
    *) echo "Usage: $0 {deps|build|load|verify|setup}" >&2; exit 2 ;;
esac
