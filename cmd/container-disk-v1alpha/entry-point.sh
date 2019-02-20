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

trap 'echo "Graceful exit"; exit 0' SIGINT SIGQUIT SIGTERM

if [ -z "$COPY_PATH" ]; then
	echo "COPY_PATH variable not set"
	exit 1
fi

if [ -n "$IMAGE_PATH" ] && [ ! -f "$IMAGE_PATH" ]; then
  echo "Image not found: $IMAGE_PATH"
  exit 1
fi

if [ -z "$IMAGE_PATH" ]; then
	IMAGE_NAME=$(ls -1 disk/ | tail -n -1)
	if [ -z "$IMAGE_NAME" ]; then
		echo "no images found in /disk directory"
		exit 1
	fi
	IMAGE_PATH=/disk/$IMAGE_NAME
fi 

IMAGE_EXTENSION=$(echo $IMAGE_PATH | sed  -n -e 's/^.*\.\(.*\)$/\1/p')

mkdir -p $COPY_PATH
echo $IMAGE_NAME | grep -q -e "raw" -e "qcow2"
if [ $? -ne 0 ]; then
	output=$(/usr/bin/qemu-img info $IMAGE_PATH --output=json)
	if [ $? -ne 0 ]; then
        echo "qemu-img info returned error"
        exit 1
    fi

    IFS=","
    echo $output | while read -r LINE; do
    #it is not a valid image if its format is not qcow2.
    if [ `echo $LINE | awk '$1 == "\"format\":" {print $1 }'` ]  && [ `echo $LINE | awk '$2 != "\"qcow2\"" {print $2 }'` ] ; then
        echo "Invalid format for image $IMAGE_PATH"
        exit 1
    fi
    #it is not a valid image if it has backing-filename
    if [ `echo $LINE | awk '$1 == "\"backing-filename\":" {print $1 }'` ]; then
        echo "Image $IMAGE_PATH is invalid because it has a backing file"
        exit 1
    fi
    done

	IMAGE_EXTENSION="raw"
	/usr/bin/qemu-img convert $IMAGE_PATH ${COPY_PATH}.${IMAGE_EXTENSION}
	if [ $? -ne 0 ]; then
		echo "Failed to convert image $IMAGE_PATH to .raw file"
		exit 1
	fi
else
	cp $IMAGE_PATH ${COPY_PATH}.${IMAGE_EXTENSION}
	if [ $? -ne 0 ]; then
		echo "Failed to copy $IMAGE_PATH to $COPY_PATH.${IMAGE_EXTENSION}"
		exit 1
	fi
fi
echo "copied $IMAGE_PATH to $COPY_PATH.${IMAGE_EXTENSION}"

touch /tmp/healthy
while [ -f "${COPY_PATH}.${IMAGE_EXTENSION}" ]; do
	sleep 5
done
