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

FROM fedora:27

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

# Create non-root user
RUN useradd -u 1001 --create-home -s /bin/bash virt-api
WORKDIR /home/virt-api
USER 1001

# Configure swagger
RUN curl -OL https://github.com/swagger-api/swagger-ui/tarball/38f74164a7062edb5dc80ef2fdddda24f3f6eb85/swagger-ui.tar.gz \
    && mkdir swagger-ui && tar xf swagger-ui.tar.gz -C swagger-ui --strip-components 1 \
    && mkdir third_party \
    && mv swagger-ui/dist third_party/swagger-ui && rm -rf swagger-ui \
    && sed -e 's@"http://petstore.swagger.io/v2/swagger.json"@"/swaggerapi/"@' -i  third_party/swagger-ui/index.html \
    && rm swagger-ui.tar.gz && rm -rf swagger-ui

COPY virt-api /virt-api

ENTRYPOINT [ "/virt-api" ]
