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

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

ENV USER_ID=1001
ENV USER_NAME=kubevirt-tests
ENV WORKSPACE=/home/${USER_NAME}
ENV DATA_DIR=${WORKSPACE}/data
ENV RESULTS_DIR=${DATA_DIR}/results
ENV TEST_MANIFESTS_DIR=${DATA_DIR}/manifests

# Create non-root user and install dependencies
RUN yum install -y findutils && \
        yum clean all && \
        useradd -u "${USER_ID}" --create-home -s /bin/bash ${USER_NAME} && \
        mkdir "${DATA_DIR}"

WORKDIR "${WORKSPACE}"
USER "${USER_ID}"

VOLUME ["${DATA_DIR}"]

ADD entrypoint.sh ${WORKSPACE}/entrypoint.sh
ADD tests.test ${WORKSPACE}/tests.test
ADD manifests ${WORKSPACE}/
ADD manifest-templator ${WORKSPACE}/

ENTRYPOINT [ "./entrypoint.sh" ]
