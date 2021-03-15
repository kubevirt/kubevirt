#!/bin/bash

ls -alZ /etc/hosts 
ls -alZ /usr/bin/virt-launcher-cap
ls -alZ /dev/termination-log
ls -alZ /etc/resolv.conf
ls -alZ /etc/hostname
ls -alZ /run/kubevirt
ls -alZ /run/kubevirt-ephemeral-disks
ls -alZ /run/kubevirt-private
ls -alZ /run/libvirt
ls -alZ /run/kubevirt/container-disks
ls -alZ /run/kubevirt/sockets
ls -alZ /run/kubevirt/hotplug-disks
mount 
strace $@