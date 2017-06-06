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

FROM kubevirt/libvirt

MAINTAINER "Stu Gott" <sgott@redhat.com>
ENV container docker

# util-linux listed here explicity as nsenter must be present in this image
RUN yum install -y \
  util-linux \
  libcgroup-tools \
  ethtool \
  sudo \
  docker && yum -y clean all

COPY qemu-kube /usr/local/bin/qemu-x86_64
RUN chmod a+x /usr/local/bin/qemu-x86_64

COPY kubevirt-sudo /etc/sudoers.d/kubevirt
RUN chmod 0640 /etc/sudoers.d/kubevirt

COPY libvirtd-limited.sh /libvirtd-limited.sh
RUN chmod a+x /libvirtd-limited.sh

# libvirtd.sh in this image differs from upstream
RUN rm -f /libvirtd.sh
COPY libvirtd.sh /libvirtd.sh
RUN chmod a+x /libvirtd.sh

CMD ["/libvirtd.sh"]
