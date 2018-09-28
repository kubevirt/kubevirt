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

trap 'echo "Graceful exit"; exit 0' SIGINT SIGQUIT SIGTERM

if [ -z "$COPY_PATH" ]; then
	echo "COPY_PATH variable not set"
	exit 1
fi

IMAGE_NAME=$(ls -1 /disk/ | tail -n 1)
if [ -z "$IMAGE_NAME" ]; then
	echo "no images found in /disk directory"
	exit 1
fi

IMAGE_PATH=/disk/$IMAGE_NAME
IMAGE_EXTENSION=$(echo $IMAGE_NAME | sed  -n -e 's/^.*\.\(.*\)$/\1/p')
IMAGE_FILE_NAME=$(basename $COPY_PATH)
IMAGE_DESTINATION_DIR=$(dirname $COPY_PATH)

function requires_conversion() {
	echo $IMAGE_NAME | grep -q -e "raw" -e "qcow2"
	return $?
}

function copy_with_conversion() {
	IMAGE_EXTENSION="raw"
	/usr/bin/qemu-img convert $IMAGE_PATH ${COPY_PATH}.${IMAGE_EXTENSION}
	if [ $? -ne 0 ]; then
		echo "Failed to convert image $IMAGE_PATH to .raw file"
		exit 1
	fi
	echo "copied $IMAGE_PATH to $COPY_PATH.${IMAGE_EXTENSION}"
}

function copy_direct() {
	cp $IMAGE_PATH ${COPY_PATH}.${IMAGE_EXTENSION}
	if [ $? -ne 0 ]; then
		echo "Failed to copy $IMAGE_PATH to $COPY_PATH.${IMAGE_EXTENSION}"
		exit 1
	fi
	echo "copied $IMAGE_PATH to $COPY_PATH.${IMAGE_EXTENSION}"
}


function bind_mount() {
	mv $IMAGE_PATH /disk/$IMAGE_FILE_NAME.${IMAGE_EXTENSION}
	mount --bind /disk $IMAGE_DESTINATION_DIR
	if [ $? -ne 0 ]; then
		echo "Failed to bind mount /disk to $IMAGE_DESTINATION_DIR"
		exit 1
	fi
	echo "bind mounted /disk to $IMAGE_DESTINATION_DIR"
}

mkdir -p $COPY_PATH
requires_conversion
if [ $? -ne 0 ]; then
	# If the image isn't in raw or qcow, conversion is required
	# which results in a copy of the image.
	copy_with_conversion
elif [ "$MOUNT_PROPAGATION" = "true" ]; then
	# If mount propagation is enabled, just bind mount the
	# disk into the destination folder.
	bind_mount
else
	# If mount propation is not enabled, simply copy the image
	# into the destination folder. 
	copy_direct
fi

touch /tmp/healthy
while [ -f "${COPY_PATH}.${IMAGE_EXTENSION}" ]; do
	sleep 5
done
