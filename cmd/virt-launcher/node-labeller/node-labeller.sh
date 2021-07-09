#!/bin/bash

set -e
# if KVM device is not present, exit immediately
if [ ! -e /dev/kvm ] && [ $(grep '\\<kvm\\>' /proc/misc | wc -l) -eq 0 ]; then
  exit 0
fi

if [ ! -e /dev/kvm ]; then
  mknod /dev/kvm c 10 $(grep '\\<kvm\\>' /proc/misc | cut -f 1 -d' ')
fi
chmod o+rw /dev/kvm

libvirtd -d

virsh domcapabilities --machine q35 --arch x86_64 --virttype kvm > /var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml

cp -r /usr/share/libvirt/cpu_map /var/lib/kubevirt-node-labeller

virsh domcapabilities --machine q35 --arch x86_64 --virttype kvm | virsh hypervisor-cpu-baseline --features /dev/stdin --machine q35 --arch x86_64 --virttype kvm > /var/lib/kubevirt-node-labeller/supported_features.xml
virsh capabilities > /var/lib/kubevirt-node-labeller/capabilities.xml