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

FROM fedora:26

# If the container is running privileged, the permissions should be
# kept under the current user.  If not, then we still need to create a
# user with the same UID in the container
ARG userid

RUN dnf -y install make git golang pkg-config libvirt-devel && \
    dnf -y clean all && \
    useradd -o -r -u $userid -U -m builder

ENV GOPATH=/builder PATH=/sbin:/bin:/builder/bin

USER $userid
WORKDIR $GOPATH/src/kubevirt.io/kubevirt/
