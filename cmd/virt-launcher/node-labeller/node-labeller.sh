#!/bin/bash

set -xeo pipefail

ARCH=$(uname -m)
MACHINE=q35
if [ "$ARCH" == "aarch64" ]; then
  MACHINE=virt
elif [ "$ARCH" != "x86_64" ]; then
  exit 0
fi

set +o pipefail

KVM_MINOR=$(grep -w 'kvm' /proc/misc | cut -f 1 -d' ')
set -o pipefail

VIRTTYPE=qemu


if [ ! -e /dev/kvm ] && [ -n "$KVM_MINOR" ]; then
  mknod /dev/kvm c 10 $KVM_MINOR
fi

if [ -e /dev/kvm ]; then
    chmod o+rw /dev/kvm
    VIRTTYPE=kvm
fi

if [ -e /dev/sev ]; then
  # QEMU requires RW access to query SEV capabilities
  chmod o+rw /dev/sev
fi

virtqemud -d

virsh domcapabilities --machine $MACHINE --arch $ARCH --virttype $VIRTTYPE > /var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml

cp -r /usr/share/libvirt/cpu_map /var/lib/kubevirt-node-labeller

# hypervisor-cpu-baseline command only works on x86
if [ "$ARCH" == "x86_64" ]; then
   virsh domcapabilities --machine $MACHINE --arch $ARCH --virttype $VIRTTYPE | virsh hypervisor-cpu-baseline --features /dev/stdin --machine $MACHINE --arch $ARCH --virttype $VIRTTYPE > /var/lib/kubevirt-node-labeller/supported_features.xml
fi

virsh capabilities > /var/lib/kubevirt-node-labeller/capabilities.xml
