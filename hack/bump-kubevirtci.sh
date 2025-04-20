#!/bin/bash
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

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

val=$(curl -L https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest)
sed -i "/^[[:blank:]]*kubevirtci_git_hash[[:blank:]]*=/s/=.*/=\"${val}\"/" hack/config-default.sh

hack/sync-kubevirtci.sh
