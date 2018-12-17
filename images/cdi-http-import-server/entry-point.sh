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

ALPINE_IMAGE_PATH=/usr/share/nginx/html/images/alpine.iso
IMAGE_PATH=/images

if [ -n "$AS_ISCSI" ]; then
    mkdir -p $IMAGE_PATH
    /usr/bin/qemu-img convert $ALPINE_IMAGE_PATH $IMAGE_PATH/alpine.raw
    if [ $? -ne 0 ]; then
        echo "Failed to convert image $ALPINE_IMAGE_PATH to .raw file"
        exit 1
    fi

    touch /tmp/healthy
    bash expose-as-iscsi.sh "${IMAGE_PATH}/alpine.raw"
fi
