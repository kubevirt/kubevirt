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

FROM fedora:27

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>
ENV container docker

RUN dnf -y install scsi-target-utils bzip2 e2fsprogs

RUN mkdir -p /images

# Add alpine image
RUN curl http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-virt-3.7.0-x86_64.iso > /images/1-alpine.img

# Cirros has been removed from the demo target for now.
# Uncomment this line if we need to add the cirros image back
# in for testing purposes.
# ADD https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img /images/2-cirros.img

ADD run-tgt.sh /

EXPOSE 3260

CMD ["/run-tgt.sh"]
