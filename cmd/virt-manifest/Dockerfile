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

FROM centos:7

# The jmliger-virt7-upstream-epel-7 has updated libvirt binaries. Since
# the statically linked libvirt-go code requires a runtime version of libvirt
# no-lower than what was used when the binary was built.
RUN yum -y install wget && \
    wget -P /etc/yum.repos.d https://copr.fedorainfracloud.org/coprs/jmliger/virt7-upstream/repo/epel-7/jmliger-virt7-upstream-epel-7.repo  \
    && yum -y install libvirt-client \
    && yum -y clean all

COPY virt-manifest /virt-manifest

ENTRYPOINT [ "/virt-manifest" ]
