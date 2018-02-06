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

FROM alpine:latest

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>
ENV container docker

RUN apk add --no-cache nfs-utils

ADD https://github.com/just-containers/s6-overlay/releases/download/v1.21.2.2/s6-overlay-amd64.tar.gz /tmp/
RUN tar xzf /tmp/s6-overlay-amd64.tar.gz -C /

COPY s6/config.init /etc/cont-init.d/00-config
COPY s6/rpcbind.run /etc/services.d/rpcbind/run
COPY s6/mountd.run /etc/services.d/mountd/run

RUN mkdir -p /volumes/cirros \
    && mkdir -p /volumes/alpine \
    && chmod -R 777 /volumes \
    && echo "/volumes *(rw,sync,no_root_squash,no_all_squash)" >> /etc/exports

# Fill NFS share with images
ADD https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img /volumes/cirros/disk.img
ADD http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-virt-3.7.0-x86_64.iso /volumes/alpine/disk.img

EXPOSE 111/tcp 111/udp 2049/tcp 2049/udp

CMD ["/init"]
