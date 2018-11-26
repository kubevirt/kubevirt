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

LABEL maintainer="David Vossel <dvossel@redhat.com>"

# Add cirros
RUN curl https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img > /disk/cirros.img
RUN mkdir -p /custom-disk && cp /disk/cirros.img /custom-disk/cirros.img
