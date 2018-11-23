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

FROM kubevirt/libvirt:4.9.0

LABEL maintainer="The KubeVirt Project <kubevirt-dev@googlegroups.com>"

RUN dnf -y install \
  socat \
  genisoimage \
  && dnf -y clean all && \
  test $(id -u qemu) = 107 # make sure that the qemu user really is 107

COPY virt-launcher /usr/bin/virt-launcher
COPY .version /.version

# Allow qemu to bind privileged ports
RUN setcap CAP_NET_BIND_SERVICE=+eip /usr/bin/qemu-system-x86_64

RUN mkdir -p /usr/share/kubevirt/virt-launcher
COPY sock-connector /usr/share/kubevirt/virt-launcher/

ENTRYPOINT [ "/usr/bin/virt-launcher" ]
