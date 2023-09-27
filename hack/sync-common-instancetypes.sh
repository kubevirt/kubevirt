#!/bin/bash

set -ex

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

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
