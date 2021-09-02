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
# Copyright 2018 Red Hat, Inc.
#

set -euo pipefail

# gracefully handle the TERM signal sent when deleting the daemonset
trap 'exit' TERM
SELINUX_TAG=$(ls -Z)

mkdir -p /images/datavolume1 /images/datavolume2 /images/datavolume3

echo "converting cirros image from qcow2 to raw, and copying it to local-storage directory, and creating a loopback device from it"
# /local-storage will be mapped to the host dir, which will also be used by the local storage provider
qemu-img convert -f qcow2 -O raw /images/cirros/disk.img /local-storage/cirros.img.raw

# Check if attached loopdevice reach limit number (100)
num=$(losetup -l | wc -l)
[ ${num} -gt 100 ] && echo "attached loopdevices have reach limit number(100)" && exit 1

# Put LOOP_DEVICE in /etc/bashrc in order to detach this loop device when the pod stopped.
LOOP_DEVICE=$(chroot /host losetup --find --show /mnt/local-storage/cirros.img.raw)
echo LOOP_DEVICE=${LOOP_DEVICE} >>/etc/bashrc
rm -f /local-storage/cirros-block-device
ln -s $LOOP_DEVICE /local-storage/cirros-block-device

echo "copy all images to host mount directory"
cp -R /images/* /hostImages/
echo "make the alpine image ready for parallel use"
cp -r /hostImages/alpine hostImages/alpine1
cp -r /hostImages/alpine hostImages/alpine2
cp -r /hostImages/alpine hostImages/alpine3
rm -rf /hostImages/alpine
echo "make the custom image ready for parallel use"
cp -r /hostImages/custom hostImages/custom1
cp -r /hostImages/custom hostImages/custom2
cp -r /hostImages/custom hostImages/custom3
rm -rf /hostImages/custom
chmod -R 777 /hostImages

# When the host is ubuntu, by default, selinux is not used, so chcon is not necessary.
# If selinux tag is set, use chcon to change /hostImages privileges.
if [ ${SELINUX_TAG:0:1} != "?" ]; then
    chcon -Rt svirt_sandbox_file_t /hostImages
fi

# Create a 4Gi blank disk image
dd if=/dev/zero of=/local-storage/hp_file.img bs=4k count=1024k
ls -al /local-storage/hp_file.img
# Put LOOP_DEVICE_HP in /etc/bashrc in order to detach this loop device when the pod stopped.
LOOP_DEVICE_HP=$(chroot /host losetup --verbose --find --show /mnt/local-storage/hp_file.img)
echo LOOP_DEVICE_HP=${LOOP_DEVICE_HP} >>/etc/bashrc
chroot /host mkfs.ext4 $LOOP_DEVICE_HP
mkdir -p /hostImages/mount_hp
ls -al /host${LOOP_DEVICE_HP}
chroot /host mount ${LOOP_DEVICE_HP} /tmp/hostImages/mount_hp
mkdir -p /host/tmp/hostImages/mount_hp/test
# When the host is ubuntu, by default, selinux is not used, so chcon is not necessary.
# If selinux tag is set, use chcon to change /hostImages privileges.
if [ ${SELINUX_TAG:0:1} != "?" ]; then
    chcon -Rt svirt_sandbox_file_t /host/tmp/hostImages/mount_hp
fi
chmod 777 /host/tmp/hostImages/mount_hp
chmod 777 /host/tmp/hostImages/mount_hp/test

cat /etc/bashrc

# for some reason without sleep, container sometime fails to create the file
sleep 10

# let the monitoring script know we're done
echo "done" >/ready

while true; do
    sleep 60
done
