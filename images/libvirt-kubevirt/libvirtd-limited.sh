#!/usr/bin/bash

set -xe

/usr/sbin/virtlogd -f /etc/libvirt/virtlogd.conf &
sleep 5

/usr/sbin/libvirtd
