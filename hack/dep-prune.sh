#!/usr/bin/env bash
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
# Copyright The KubeVirt Authors.
#

set -ex

source hack/common.sh

# create symbolic link on client-go package to avoid duplication
rm -rf ${KUBEVIRT_DIR}/vendor/kubevirt.io/client-go
mkdir -p ${KUBEVIRT_DIR}/vendor/kubevirt.io
ln -s ../../staging/src/kubevirt.io/client-go/ ${KUBEVIRT_DIR}/vendor/kubevirt.io/client-go

# create symbolic link on api package to avoid duplication
rm -rf ${KUBEVIRT_DIR}/vendor/kubevirt.io/api
mkdir -p ${KUBEVIRT_DIR}/vendor/kubevirt.io
ln -s ../../staging/src/kubevirt.io/api/ ${KUBEVIRT_DIR}/vendor/kubevirt.io/api
