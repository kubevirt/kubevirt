#!/bin/bash
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
source hack/version.sh
kubevirt::version::get_version_vars

RPM_OUT_DIR="${OUT_DIR}/rpmbuild"
SOURCE="kubevirt-${KUBEVIRT_GIT_COMMIT}"
NUM_VERSION=$(echo ${KUBEVIRT_GIT_VERSION} | awk -F '-' {'print $1'} | tr -d '[:alpha:]')
RELEASE=$(echo ${KUBEVIRT_GIT_VERSION} | awk -F '-' {'print $2'})
TARBALL="kubevirt-${NUM_VERSION}.tar.gz"

mkdir -p ${RPM_OUT_DIR}/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS}
rsync -a --delete --filter=":- .gitignore" . ${RPM_OUT_DIR}/SOURCES/${SOURCE}
tar -czf ${RPM_OUT_DIR}/SOURCES/${TARBALL} -C ${RPM_OUT_DIR}/SOURCES ${SOURCE}
rpmbuild --define "_topdir ${RPM_OUT_DIR}" \
    --define "commit ${KUBEVIRT_GIT_COMMIT}" \
    --define "version ${NUM_VERSION}" \
    --define "release ${RELEASE}" \
    --define "kubevirt_git_version ${KUBEVIRT_GIT_VERSION}" \
    -ba ${KUBEVIRT_DIR}/kubevirt.spec
