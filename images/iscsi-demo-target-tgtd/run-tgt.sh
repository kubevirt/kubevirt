#!/bin/bash
# https://fedoraproject.org/wiki/Scsi-target-utils_Quickstart_Guide

set -e

SIZE=${SIZE:-1GB}
WWN=${WWN:-iqn.2017-01.io.kubevirt:sn.42}

[[ -f /volume/0-custom.img ]] || truncate -s ${SIZE} /volume/0-custom.img

echo "Starting tgtd"
tgtd -f &
sleep 2

echo "Adding target and exposing it"
tgtadm --lld iscsi --mode target --op new --tid=1 --targetname $WWN
tgtadm --lld iscsi --mode target --op bind --tid=1 -I ALL

LUNID=1
add_lun_for_file() {
  local FN=$1
  tgtadm --lld iscsi --mode logicalunit --op new --tid=1 --lun=$LUNID -b $FN
  tgtadm --lld iscsi --mode logicalunit --op update --tid=1 --lun=$LUNID --params thin_provisioning=1
  [[ "$FN" = *.iso ]] && tgtadm --lld iscsi --mode logicalunit --op update --tid=1 --lun=$LUNID --params readonly=1
  LUNID=$(( $LUNID + 1 ))
}

echo "Adding every file in /volume as a LUN"
for F in $(ls -1 /volume/* | sort) ; do
  echo "- Adding LUN ID $LUNID for file '$F'"
  add_lun_for_file $F
done

echo "Adding listed host paths ($EXPORT_HOST_PATHS)"
for P in $EXPORT_HOST_PATHS ; do
  F=/host/$P
  echo "- Adding LUN ID $LUNID for file '$F'"
  add_lun_for_file $F
done

echo "Start monitoring"
touch previous_state
while true ; do
  tgtadm --lld iscsi --mode target --op show > current_state
  diff -q previous_state current_state || ( date ; cat current_state ; )
  mv -f current_state previous_state
  sleep 3
done

# vim: et tw=2:
