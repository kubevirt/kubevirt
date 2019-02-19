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

FROM fedora:28

LABEL maintainer="The KubeVirt Project <kubevirt-dev@googlegroups.com>"
ENV container docker

RUN dnf install -y nginx qemu-guest-agent qemu-img scsi-target-utils \
    && dnf -y clean all

RUN mkdir -p /usr/share/nginx/html/images \
    && curl http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-virt-3.7.0-x86_64.iso > /usr/share/nginx/html/images/alpine.iso \
    && curl https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img > /usr/share/nginx/html/images/cirros.img \
    && rm -f /etc/nginx/nginx.conf

RUN cp /usr/bin/qemu-ga /usr/share/nginx/html/

ADD nginx.conf /etc/nginx/

EXPOSE 80

ADD entry-point.sh /
ADD expose-as-iscsi.sh /

CMD ["/entry-point.sh"]
