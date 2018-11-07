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
ENV container docker

RUN dnf -y install make git gcc && dnf -y clean all

ENV GIMME_GO_VERSION=1.9.2
RUN mkdir -p /gimme && curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=/gimme bash >> /etc/profile.d/gimme.sh

ENV GOPATH="/go" GOBIN="/usr/bin"
RUN \
    mkdir -p /go && \
    source /etc/profile.d/gimme.sh && \
    go get github.com/masterzen/winrm-cli
