#!/bin/bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2017 Red Hat, Inc.
#

# https://fedoraproject.org/wiki/Scsi-target-utils_Quickstart_Guide

set -e

SIZE=${SIZE:-1GB}
WWN=${WWN:-iqn.2017-01.io.kubevirt:sn.42}

mkdir -p /volumes
mkdir -p /mnt/disk

[[ -f /images/0-custom.img ]] || truncate -s ${SIZE} /images/0-custom.img

echo "Starting tgtd"
tgtd -f &
sleep 5

# ensure tgtd is bound to port 3260
lsof -Pan -p $(pgrep tgtd) -i | grep 3260 -q

echo "Adding target and exposing it"
tgtadm --lld iscsi --mode target --op new --tid=1 --targetname $WWN
tgtadm --lld iscsi --mode target --op bind --tid=1 -I ALL

if [ -n "$PASSWORD" ] && [ -n "$USERNAME" ]; then
    tgtadm --lld iscsi --op new --mode account --user $USERNAME --password $PASSWORD
    tgtadm --lld iscsi --op bind --mode account --tid=1 --user $USERNAME
fi

LUNID=1

convert_to_filesystem() {
    local FN=$1
    local DISK_NAME="/tmp/$(basename $FN)"
    dd if=/dev/zero of=$DISK_NAME bs=1M count=1024
    mkfs.ext4 $DISK_NAME
    sleep 10
    mount -o loop,rw,sync $DISK_NAME /mnt/disk
    mv $FN /mnt/disk/disk.img
    umount /mnt/disk
    mv $DISK_NAME /volumes
}

echo "Convert every file in /images to filesystem disk"
for F in $(ls -1 /images/* | sort); do
    echo "- Convert file '$F'"
    convert_to_filesystem $F
done

add_lun_for_file() {
    local FN=$1
    tgtadm --lld iscsi --mode logicalunit --op new --tid=1 --lun=$LUNID -b $FN
    tgtadm --lld iscsi --mode logicalunit --op update --tid=1 --lun=$LUNID --params thin_provisioning=1
    LUNID=$(($LUNID + 1))
}

echo "Adding every file in /volume as a LUN"
for F in $(ls -1 /volumes/* | sort); do
    echo "- Adding LUN ID $LUNID for file '$F'"
    add_lun_for_file $F
done

echo "Adding listed host paths ($EXPORT_HOST_PATHS)"
for P in $EXPORT_HOST_PATHS; do
    F=/host/$P
    echo "- Adding LUN ID $LUNID for file '$F'"
    add_lun_for_file $F
done

echo "Start monitoring"
touch /tmp/previous_state
while true; do
    tgtadm --lld iscsi --mode target --op show >current_state
    diff -q /tmp/previous_state current_state || (
        date
        cat current_state
    )
    mv -f current_state /tmp/previous_state
    sleep 3
done

# vim: et ts=2:
