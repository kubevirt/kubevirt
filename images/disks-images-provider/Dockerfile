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

FROM fedora:28

LABEL maintainer="The KubeVirt Project <kubevirt-dev@googlegroups.com>"
ENV container docker

# Prepare all images
RUN yum -y install qemu-img \
    && mkdir -p /images/custom /images/alpine /images/cirros /images/datavolume1 /images/datavolume2 /images/datavolume3 \
    && truncate -s 64M /images/custom/disk.img \
    && curl http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-virt-3.7.0-x86_64.iso > /images/alpine/disk.img \
    && curl https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img > /images/cirros/disk.img

ADD entrypoint.sh /

CMD ["/entrypoint.sh"]
