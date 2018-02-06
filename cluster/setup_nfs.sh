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
set -ex

NFS_SHARE_DIR=/var/nfsshare

yum install -y nfs-utils

mkdir $NFS_SHARE_DIR
mkdir $NFS_SHARE_DIR/cirros
mkdir $NFS_SHARE_DIR/alpine
chmod -R 755 $NFS_SHARE_DIR
chown -R nfsnobody:nfsnobody $NFS_SHARE_DIR

echo "$NFS_SHARE_DIR 192.168.0.0/16(rw,sync,no_root_squash,no_all_squash)" >>/etc/exports

systemctl enable rpcbind nfs-server
systemctl start rpcbind nfs-server

# Fill NFS share with images
curl \
    https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img \
    >$NFS_SHARE_DIR/cirros/disk.img

curl \
    http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-virt-3.7.0-x86_64.iso \
    >$NFS_SHARE_DIR/alpine/disk.img
