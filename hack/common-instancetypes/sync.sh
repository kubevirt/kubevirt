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

source $(dirname "$0")/default.sh

_common_instancetypes_base_url="https://github.com/kubevirt/common-instancetypes/releases/download"
_cluster_instancetypes_path="pkg/virt-operator/resource/generate/components/data/common-clusterinstancetypes-bundle.yaml"
_cluster_preferences_path="pkg/virt-operator/resource/generate/components/data/common-clusterpreferences-bundle.yaml"

curl \
    -L "${_common_instancetypes_base_url}/${common_instancetypes_version}/common-clusterinstancetypes-bundle-${common_instancetypes_version}.yaml" \
    -o "${_cluster_instancetypes_path}"
echo "${cluster_instancetypes_sha256} ${_cluster_instancetypes_path}" | sha256sum --check --strict

curl \
    -L "${_common_instancetypes_base_url}/${common_instancetypes_version}/common-clusterpreferences-bundle-${common_instancetypes_version}.yaml" \
    -o "${_cluster_preferences_path}"
echo "${cluster_preferences_sha256} ${_cluster_preferences_path}" | sha256sum --check --strict
