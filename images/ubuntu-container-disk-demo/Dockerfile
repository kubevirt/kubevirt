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

FROM kubevirt/registry-disk-v1alpha

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

# Add ubuntu disk image
RUN curl -g -L https://cloud-images.ubuntu.com/cosmic/current/cosmic-server-cloudimg-amd64.img > /disk/ubuntu.qcow2
RUN md5sum /disk/ubuntu.qcow2 | grep $(curl https://cloud-images.ubuntu.com/cosmic/current/MD5SUMS | grep cosmic-server-cloudimg-amd64.img | cut -f1 -d' ')
