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
# Copyright 2018 Red Hat, Inc.
#

FROM fedora:28

LABEL maintainer="The KubeVirt Project <kubevirt-dev@googlegroups.com>"

# Create non-root user
RUN useradd -u 1001 --create-home -s /bin/bash virtctl
WORKDIR /home/virtctl
USER 1001
COPY subresource-access-test /subresource-access-test

ENTRYPOINT [ "/subresource-access-test" ]
