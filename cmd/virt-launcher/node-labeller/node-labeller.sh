#!/bin/bash

set -e

if [ ! -e /dev/kvm ]; then
  mknod /dev/kvm c 10 $(grep '\\<kvm\\>' /proc/misc | cut -f 1 -d' ')
fi
chmod o+rw /dev/kvm

libvirtd -d

virsh domcapabilities --machine q35 --arch x86_64 --virttype kvm > /var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml

cp -r /usr/share/libvirt/cpu_map /var/lib/kubevirt-node-labeller
