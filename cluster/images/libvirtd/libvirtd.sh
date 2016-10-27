#!/usr/bin/bash

set -xe

fatal() { echo "FATAL: $@" >&2 ; exit 2 ; }
[[ -f /host/var/run/libvirtd.pid ]] && fatal "libvirtd seems to be running on the host"
brctl show | grep virbr >/dev/null && fatal "libvirtd bridges are present on the host"

KVM_GID=$(grep "kvm" /etc/group | awk 'BEGIN {FS=":"}; {print $3};')

# HACK
# Use hosts's /dev to see new devices and allow macvtap
mkdir /odev && {
  # the kvm group inside this container is 36--make sure that libvirtd/qemu can
  # use the kvm device using that
  # Note: this is done before moving the original /dev directory
  chown root:"$KVM_GID" /dev/kvm
  chmod g+rw /dev/kvm
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
