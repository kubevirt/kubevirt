#! /bin/bash

set -ex

function configure_vf_driver() {
    vf_sys_device=$1
    driver=$2

    vf_pci_address=$(basename $vf_sys_device)
    # Check if a VF is bound to a diffrent driver
    if [ -d "$vf_sys_device/driver" ]; then
        vf_bus_pci_device_driver=$(readlink -e $vf_sys_device/driver)
        vf_driver_name=$(basename $vf_bus_pci_device_driver)

        # Check if VF alteady configured with supported driver
        if [[ $vf_driver_name == $driver ]]; then
            return
        else
            echo "Unbind VF $vf_pci_address from $vf_driver_name driver"
            echo $vf_pci_address >> $vf_bus_pci_device_driver/unbind
        fi
    fi

    echo "Bind VF $vf_pci_address to $driver driver"
    echo $driver >> $vf_sys_device/driver_override

    echo $vf_pci_address >> /sys/bus/pci/drivers/$driver/bind

    echo "" >> $vf_sys_device/driver_override

    return 0
}

function configure_pf_vf_count() {
    pf_net_device=$1
    vfs_count=$2

    pf_name=$(basename $pf_net_device)
    pf_sys_device=$(readlink -e $pf_net_device)

    sriov_totalvfs_content=$(cat $pf_sys_device/sriov_totalvfs)
    if [ $sriov_totalvfs_content -lt $vfs_count ]; then
        echo "VF's count should be up to sriov_totalvfs"
        return 1
    fi

    sriov_numvfs_content=$(cat $pf_sys_device/sriov_numvfs)
    if [ $sriov_numvfs_content -ne $vfs_count ]; then
        echo "Creating $vfs_count VF's on PF $pf_name"
        echo 0 >> $pf_sys_device/sriov_numvfs
        echo $vfs_count >> $pf_sys_device/sriov_numvfs
        sleep 3
    fi

    return 0
}

SUPPORTED_DRIVER="vfio-pci"

if [ "$(id -u)" -ne 0 ]; then
  echo "This script requires sudo privileges"
  exit 1
fi

sysfs_permissions=$(grep -Po 'sysfs.*\K(ro|rw)' /proc/mounts)
if [ $sysfs_permissions != rw ]; then
  echo "sysfs is read-only, try to remount as RW"
  exit 1
fi

pfs=$(ls /sys/class/net/*/device/sriov_totalvfs)
for pf in $pfs; do
    echo "Create VF's"
    pf_device=$(dirname $pf)
    sriov_numvfs=$(cat $pf_device/sriov_totalvfs)
    configure_pf_vf_count $pf_device $sriov_numvfs

    echo "Configuring VF's drivers"
    # /sys/class/net/<pf name>/device/virtfn*
    vfs_sys_devices=$(readlink -e $pf_device/virtfn*)
    set +e
    for vf in $vfs_sys_devices; do
        ls -l  $vf/driver
        configure_vf_driver $vf $SUPPORTED_DRIVER
    done
    set -e

    echo "###### bebug"
    ip link show
    for vf in $vfs_sys_devices; do
        ls -l  $vf/driver
    done
done