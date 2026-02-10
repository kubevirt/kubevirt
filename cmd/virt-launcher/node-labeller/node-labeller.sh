#!/bin/bash

set -xeo pipefail

# Default values for env vars, can be overridden by user input
KVM_HYPERVISOR_DEVICE="kvm"
KVM_VIRTTYPE="kvm"

if [ -z "$HYPERVISOR_DEVICE" ] || [ -z "$PREFERRED_VIRTTYPE" ]; then
    echo "Warning: Env vars HYPERVISOR_DEVICE or PREFERRED_VIRTTYPE not set. Defaulting to KVM values for both vars"
    echo "Currently specified values: HYPERVISOR_DEVICE='$HYPERVISOR_DEVICE', PREFERRED_VIRTTYPE='$PREFERRED_VIRTTYPE'"
    HYPERVISOR_DEVICE="$KVM_HYPERVISOR_DEVICE"
    PREFERRED_VIRTTYPE="$KVM_VIRTTYPE"
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
