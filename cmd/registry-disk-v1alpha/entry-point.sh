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

PORT=${PORT:-3260}
WWN=${WWN:-iqn.2017-01.io.kubevirt:wrapper}
LUNID=1
IMAGE_NAME=$(ls -1 /disk/ | tail -n 1)
IMAGE_PATH=/disk/$IMAGE_NAME

if [ -n "$PASSWORD_BASE64" ]; then
	PASSWORD=$(echo $PASSWORD_BASE64 | base64 -d)
fi
if [ -n "$USERNAME_BASE64" ]; then
	USERNAME=$(echo $USERNAME_BASE64 | base64 -d)
fi

# If PASSWORD is provided, enable authentication features
authenticate=0
if [ -n "$PASSWORD" ]; then
	authenticate=1
fi

if [ -z "$IMAGE_NAME" ] || ! [ -f "$IMAGE_PATH" ]; then
	echo "vm image not found in /disk directory"
	exit 1
fi

echo $IMAGE_NAME | grep -q "\.raw$"
if [ $? -ne 0 ]; then
	/usr/bin/qemu-img convert $IMAGE_PATH /disk/image.raw
	if [ $? -ne 0 ]; then
		echo "Failed to convert image $IMAGE_PATH to .raw file"
		exit 1
	fi
	IMAGE_PATH=/disk/image.raw
fi

# USING 'set -e' error detection for everything below this point.
set -e

echo "Starting tgtd at port $PORT"
tgtd -f --iscsi portal="0.0.0.0:${PORT}" &
sleep 5

echo "Adding target and exposing it"
tgtadm --lld iscsi --mode target --op new --tid=1 --targetname $WWN
tgtadm --lld iscsi --mode target --op bind --tid=1 -I ALL

if [ $authenticate -eq 1 ]; then
	echo "Adding authentication for user $USERNAME"
	tgtadm --lld iscsi --op new --mode account --user $USERNAME --password $PASSWORD
	tgtadm --lld iscsi --op bind --mode account --tid=1 --user $USERNAME
fi

echo "Adding volume file as LUN"
tgtadm --lld iscsi --mode logicalunit --op new --tid=1 --lun=$LUNID -b $IMAGE_PATH
tgtadm --lld iscsi --mode logicalunit --op update --tid=1 --lun=$LUNID --params thin_provisioning=1

echo "Start monitoring"
touch /tmp/healthy
touch previous_state
while true ; do
	tgtadm --lld iscsi --mode target --op show > current_state
	diff -q previous_state current_state || ( date ; cat current_state ; )
	mv -f current_state previous_state
	sleep 5
done
