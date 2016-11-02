#!/usr/bin/bash

set -xe

# HACK
# Use hosts's /dev to see new devices and allow macvtap
mkdir /odev && {
  mount --rbind /dev /odev
  mount --rbind /host/dev /dev
  # Keep some devices from the original /dev
  mount --rbind /odev/shm /dev/shm
  mount --rbind /odev/mqueue /dev/mqueue
  # Keep ptmx/pts for pty creation
  mount --rbind /odev/pts /dev/pts
  mount --rbind /dev/pts/ptmx /dev/ptmx
}

/usr/sbin/libvirtd
