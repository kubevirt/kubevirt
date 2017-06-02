#
# This file is part of the kubevirt project
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

FROM debian:sid

MAINTAINER "Fabian Deutsch" \<fabiand@redhat.com\>
ENV container docker

RUN apt-get update && apt-get install -y \
  tgt bash curl bzip2 qemu-utils

RUN mkdir -p /volume

# Add alpine image
RUN curl \
      https://nl.alpinelinux.org/alpine/v3.5/releases/x86_64/alpine-virt-3.5.1-x86_64.iso \
      > /volume/alpine.iso

# Add cirros
RUN curl \
      http://download.cirros-cloud.net/0.3.5/cirros-0.3.5-x86_64-disk.img \
      > /volume/cirros.img \
      && qemu-img convert -O raw /volume/cirros.img /volume/cirros.rawa \
      && rm volume/cirros.img

ADD run-tgt.sh /

EXPOSE 3260

CMD ["/run-tgt.sh"]
