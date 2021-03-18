#!/bin/bash

echo hosts
ls -alZ /etc/hosts 
echo cap
ls -alZ /usr/bin/virt-launcher-cap
echo term-log
ls -alZ /dev/termination-log
echo resolv
ls -alZ /etc/resolv.conf
echo hostname 
ls -alZ /etc/hostname
echo kv
ls -alZ /run/kubevirt
ls -alZ /var/run/kubevirt
echo e-disk
ls -alZ /run/kubevirt-ephemeral-disks
echo kv-private
ls -alZ /run/kubevirt-private
ls -alZ /var/run/kubevirt-private
echo libvirt
ls -alZ /run/libvirt
echo cdisk
ls -alZ /run/kubevirt/container-disks
ls -alZ /run/kubevirt/sockets
ls -alZ /run/kubevirt/hotplug-disks
echo lib
ls -alZ /lib64
mount 
strace $@