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

# builder
FROM pkotas/golang-centos:latest as builder

RUN yum -y update &&\
    yum -y install \
        gcc \
        libvirt-client \
        libvirt-dev 

RUN mkdir -p /usr/src/build/src/kubevirt.io/kubevirt && \
    mkdir -p /usr/src/build/pkg && \ 
    mkdir -p /usr/src/build/bin

ENV GOPATH=/usr/src/build

WORKDIR /usr/src/build/src
COPY vendor .
COPY . kubevirt.io/kubevirt/

WORKDIR kubevirt.io/kubevirt/cmd/virt-controller
RUN go build virt-controller.go

# production
FROM centos:latest 
LABEL maintainer="The KubeVirt Project <kubevirt-dev@googlegroups.com>"

# Create non-root user
RUN useradd -u 1001 --create-home -s /bin/bash virt-controller
USER 1001

WORKDIR /usr/local/bin

COPY --from=builder /usr/src/build/src/kubevirt.io/kubevirt/cmd/virt-controller/virt-controller .

ENTRYPOINT [ "./virt-controller"]
