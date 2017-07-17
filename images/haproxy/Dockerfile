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

FROM haproxy:1.6-alpine

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

COPY haproxy.cfg /usr/local/etc/haproxy/haproxy.cfg

RUN sed -i "s|/run|/haproxy/run|g" /docker-entrypoint.sh
RUN cp /docker-entrypoint.sh /docker-entrypoint-orig.sh

COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN addgroup -S haproxy && adduser -S -D -h /haproxy -s /bin/false -G haproxy -g haproxy haproxy
USER haproxy
RUN mkdir /haproxy/run
