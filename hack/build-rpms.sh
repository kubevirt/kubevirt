#!/usr/bin/env bash
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

set -e

source hack/common.sh
source hack/config.sh

if [ X"$1" == "XDOCKERIZED" ]; then
    echo ${csv_version}
    rm -rf ${RPMS_OUT_DIR}

    TMP_DIR=$(mktemp -d)
    cleanup() {
        ret=$?
        rm -rf "${TMP_DIR}"
        exit ${ret}
    }
    trap "cleanup" INT TERM EXIT

    RPM_TMPDIR=${TMP_DIR}/rpmbuild
    BUILD_DIR=${RPM_TMPDIR}/BUILD

    mkdir -p ${RPM_TMPDIR}/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS}
    cp -v ${KUBEVIRT_DIR}/LICENSE ${BUILD_DIR}
    cp -v ${KUBEVIRT_DIR}/cmd/virt-handler/*.cil ${BUILD_DIR}
    cp ${KUBEVIRT_DIR}/cmd/virt-handler/kubevirt-selinux.spec.in ${RPM_TMPDIR}/SPECS/kubevirt-selinux.spec

    sed -i "s/{{.CsvVersion}}/${csv_version}/" ${RPM_TMPDIR}/SPECS/kubevirt-selinux.spec

    rpmbuild -bb --define "_topdir ${RPM_TMPDIR}" ${RPM_TMPDIR}/SPECS/kubevirt-selinux.spec

    mkdir -p ${RPMS_OUT_DIR}
    cp ${RPM_TMPDIR}/RPMS/noarch/kubevirt-selinux-*.noarch.rpm ${RPMS_OUT_DIR}
else
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/build-rpms.sh DOCKERIZED"
fi
