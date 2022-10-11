#!/bin/bash

set -e

KVM_MINOR=$(grep -w 'kvm' /proc/misc | cut -f 1 -d' ')
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

libvirtd -d

CPU_MODEL=`virsh nodeinfo|grep 'CPU model'|awk -F: '{print $2}'|xargs`

cp -r /usr/share/libvirt/cpu_map /var/lib/kubevirt-node-labeller

if [[ "$CPU_MODEL" == "aarch64" ]]; then
    virsh domcapabilities --machine virt --arch aarch64 --virttype $VIRTTYPE > /var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml
    > /var/lib/kubevirt-node-labeller/supported_features.xml
    sed -i "s/<mode name='host-model' supported='no'\/>/<mode name='host-model' supported='yes'>\n<model fallback='forbid'>Kunpeng<\/model>\n<vendor>Kunpeng<\/vendor>\n<\/mode>/g" /var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml
else
    virsh domcapabilities --machine q35 --arch x86_64 --virttype $VIRTTYPE > /var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml
    virsh domcapabilities --machine q35 --arch x86_64 --virttype $VIRTTYPE | virsh hypervisor-cpu-baseline --features /dev/stdin --machine q35 --arch x86_64 --virttype $VIRTTYPE > /var/lib/kubevirt-node-labeller/supported_features.xml
fi

virsh capabilities > /var/lib/kubevirt-node-labeller/capabilities.xml
