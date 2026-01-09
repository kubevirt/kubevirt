#!/bin/bash

set -xeo pipefail

while getopts d:t: flag; do
    case "${flag}" in
    d) HYPERVISOR_DEVICE=${OPTARG} ;;
    t) PREFERRED_VIRTTYPE=${OPTARG} ;;
    *)
        echo "Invalid option"
        exit 1
        ;;
    esac
done

if [ -z "$HYPERVISOR_DEVICE" ] || [ -z "$PREFERRED_VIRTTYPE" ]; then
    echo "Error: Missing required arguments."
    echo "Usage: $0 -d <HYPERVISOR_DEVICE> -t <PREFERRED_VIRTTYPE>"
    exit 1
fi

ARCH=$(uname -m)
MACHINE=q35
if [ "$ARCH" == "aarch64" ]; then
  MACHINE=virt
elif [ "$ARCH" == "s390x" ]; then
  MACHINE=s390-ccw-virtio
elif [ "$ARCH" != "x86_64" ]; then
  exit 0
fi

set +o pipefail

HYPERVISOR_DEV_PATH="/dev/${HYPERVISOR_DEVICE}"
HYPERVISOR_DEV_MINOR=$(grep -w ${HYPERVISOR_DEVICE} /proc/misc | cut -f 1 -d' ')
set -o pipefail

VIRTTYPE=qemu

if [ ! -e "$HYPERVISOR_DEV_PATH" ] && [ -n "$HYPERVISOR_DEV_MINOR" ]; then
  mknod "$HYPERVISOR_DEV_PATH" c 10 "$HYPERVISOR_DEV_MINOR"
fi

if [ -e "$HYPERVISOR_DEV_PATH" ]; then
    chmod o+rw "$HYPERVISOR_DEV_PATH"
    VIRTTYPE=${PREFERRED_VIRTTYPE}
fi

if [ -e /dev/sev ]; then
  # QEMU requires RW access to query SEV capabilities
  chmod o+rw /dev/sev
fi

virtqemud -d

virsh domcapabilities --machine $MACHINE --arch $ARCH --virttype $VIRTTYPE > /var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml

# hypervisor-cpu-baseline command only works on x86 and s390x
if [ "$ARCH" == "x86_64" ] || [ "$ARCH" == "s390x" ]; then
   virsh domcapabilities --machine $MACHINE --arch $ARCH --virttype $VIRTTYPE | virsh hypervisor-cpu-baseline --features /dev/stdin --machine $MACHINE --arch $ARCH --virttype $VIRTTYPE > /var/lib/kubevirt-node-labeller/supported_features.xml
fi

virsh capabilities > /var/lib/kubevirt-node-labeller/capabilities.xml
