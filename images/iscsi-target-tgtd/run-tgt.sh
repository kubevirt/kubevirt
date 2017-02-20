#!/bin/bash
# https://fedoraproject.org/wiki/Scsi-target-utils_Quickstart_Guide

set -e

die() { echo "ERR: $@" ; exit 2 ; }

SIZE_MB=${SIZE:-1024}
WWN=iqn.2017-01.io.kubevirt:sn.42

if [[ "$GENERATE_DEMO_OS_SEED" == "alpine" ]]; then
  echo "Creating alpine demo OS image as requested"
  mkdir -p /volume
  curl -o /volume/file.img https://nl.alpinelinux.org/alpine/v3.5/releases/x86_64/alpine-virt-3.5.1-x86_64.iso
elif [[ -n "$GENERATE_DEMO_OS_SEED" ]]; then
  echo "Creating qemu demo OS image as requested"
  mkdir -p /volume
  curl http://download.qemu-project.org/linux-0.2.img.bz2 | bunzip2 > /volume/file.img
else
  # Otherwise do the usual checks
  echo "Checking volume"
  [[ -d /volume ]] || die "No persistent volume provided"
  [[ -f /volume/file.img ]] || truncate -s ${SIZE} /volume/file.img
fi

echo "Starting tgtd"
tgtd -f &
sleep 2

echo "Adding target and LUN"
tgtadm --lld iscsi --mode target --op new --tid=1 --targetname $WWN
tgtadm --lld iscsi --mode logicalunit --op new --tid 1 --lun 1 -b /volume/file.img
tgtadm --lld iscsi --mode target --op bind --tid 1 -I ALL

echo "Start monitoring"
while true ; do
  date
  tgtadm --lld iscsi --mode target --op show
  sleep 3
done
