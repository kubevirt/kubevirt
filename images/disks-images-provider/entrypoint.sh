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

echo "converting cirros image from qcow2 to raw, and copying it to local-storage directory, and creating a loopback device from it"
# /local-storage will be mapped to the host dir, which will also be used by the local storage provider
qemu-img convert -f qcow2 -O raw /images/cirros/disk.img /local-storage/cirros.img.raw
LOOP_DEVICE=$(losetup --find --show /local-storage/cirros.img.raw)
rm -f /local-storage/cirros-block-device
ln -s $LOOP_DEVICE /local-storage/cirros-block-device

echo "copy all images to host mount directory"
cp -R /images/* /hostImages/

# for some reason without sleep, container sometime fails to create the file
sleep 10

# let the monitoring script know we're done
echo "done" >/ready

while true; do
    sleep 60
done
