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

WORKDIR kubevirt.io/kubevirt/cmd/virt-api
RUN go build virt-api.go

# production
FROM centos:latest 
LABEL maintainer="The KubeVirt Project <kubevirt-dev@googlegroups.com>"

# Create non-root user
RUN useradd -u 1001 --create-home -s /bin/bash virt-api
USER 1001

WORKDIR /usr/local/bin

# Configure swagger
RUN curl -OL https://github.com/swagger-api/swagger-ui/tarball/38f74164a7062edb5dc80ef2fdddda24f3f6eb85/swagger-ui.tar.gz \
    && mkdir swagger-ui && tar xf swagger-ui.tar.gz -C swagger-ui --strip-components 1 \
    && mkdir third_party \
    && mv swagger-ui/dist third_party/swagger-ui && rm -rf swagger-ui \
    && sed -e 's@"http://petstore.swagger.io/v2/swagger.json"@"/swaggerapi/"@' -i  third_party/swagger-ui/index.html \
    && rm swagger-ui.tar.gz && rm -rf swagger-ui


COPY --from=builder /usr/src/build/src/kubevirt.io/kubevirt/cmd/virt-api/virt-api .

ENTRYPOINT [ "./virt-api"]
