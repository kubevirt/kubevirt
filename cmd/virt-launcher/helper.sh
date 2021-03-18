#!/bin/bash

echo hosts
ls -ialZ /etc/hosts 
echo cap
ls -ialZ /usr/bin/virt-launcher-cap
echo term-log
ls -ialZ /dev/termination-log
echo resolv
ls -ialZ /etc/resolv.conf
echo hostname 
ls -ialZ /etc/hostname
echo kv
ls -ialZ /run/kubevirt
ls -ialZ /var/run/kubevirt
echo e-disk
ls -ialZ /run/kubevirt-ephemeral-disks
echo kv-private
ls -ialZ /run/kubevirt-private
ls -ialZ /var/run/kubevirt-private
echo libvirt
ls -ialZ /run/libvirt
echo cdisk
ls -ialZ /run/kubevirt/container-disks
ls -ialZ /run/kubevirt/sockets
ls -ialZ /run/kubevirt/hotplug-disks
echo lib
ls -ialZ /lib64
echo proc
ls -ialZ /proc
echo dev
ls -ialZ /dev
ps axZ
mount 
strace $@ 