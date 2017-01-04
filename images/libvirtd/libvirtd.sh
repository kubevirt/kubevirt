#!/usr/bin/bash

set -xe

fatal() { echo "FATAL: $@" >&2 ; exit 2 ; }
[[ -f /host/var/run/libvirtd.pid ]] && fatal "libvirtd seems to be running on the host"
brctl show | grep virbr >/dev/null && fatal "libvirtd bridges are present on the host"

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
  # Use the original /dev/kvm to ensure group ownership is honored
  mount --rbind /odev/kvm /dev/kvm
}

/usr/sbin/virtlogd &
/usr/sbin/libvirtd -l
