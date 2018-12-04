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

FROM kubevirt/container-disk-v1alpha

LABEL maintainer="The KubeVirt Project <kubevirt-dev@googlegroups.com>"

RUN curl https://fedorapeople.org/groups/virt/virtio-win/virtio-win.repo -o /etc/yum.repos.d/virtio-win.repo \
	&& dnf install -y virtio-win \
	&& dnf clean all \
	&& mkdir -p /disk \
	&& cp -L /usr/share/virtio-win/virtio-win.iso /disk/virtio-win.iso \
	&& rm -R /usr/share/virtio-win
