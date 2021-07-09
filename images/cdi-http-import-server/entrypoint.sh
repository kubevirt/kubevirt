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
set -x

trap 'echo "Graceful exit"; exit 0' SIGINT SIGQUIT SIGTERM

ALPINE_IMAGE_PATH=/usr/share/nginx/html/images/alpine.iso
CIRROS_IMAGE_PATH=/usr/share/nginx/html/images/cirros.img
FEDORA_IMAGE_PATH=/usr/share/nginx/html/images/fedora.img
IMAGE_PATH=/images
IMAGE_NAME=${IMAGE_NAME:-cirros}

case "$IMAGE_NAME" in
cirros) CONVERT_PATH=$CIRROS_IMAGE_PATH ;;
alpine) CONVERT_PATH=$ALPINE_IMAGE_PATH ;;
fedora-cloud) CONVERT_PATH=$FEDORA_IMAGE_PATH ;;
fedora-with-test-tooling) CONVERT_PATH=$FEDORA_IMAGE_PATH ;;
*)
    echo "failed to find image $IMAGE_NAME"
    ;;
esac

# use as ISCSI target
if [ -n "$AS_ISCSI" ]; then
    mkdir -p $IMAGE_PATH
    /usr/bin/qemu-img convert -O raw $CONVERT_PATH $IMAGE_PATH/disk.raw
    if [ $? -ne 0 ]; then
        echo "Failed to convert image $CONVERT_PATH to .raw file"
        exit 1
    fi

    touch /tmp/healthy
    bash expose-as-iscsi.sh "${IMAGE_PATH}/disk.raw"
    # use as nginx server
elif [ -n "$AS_EMPTY" ]; then
    mkdir -p $IMAGE_PATH
    dd if=/dev/zero of=$IMAGE_PATH/disk.raw bs=1M count=1024
    /usr/sbin/mkfs.ext4 $IMAGE_PATH/disk.raw
    sleep 20
    touch /tmp/healthy
    bash expose-as-iscsi.sh "${IMAGE_PATH}/disk.raw"
else
    /usr/sbin/nginx
fi
