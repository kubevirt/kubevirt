#!/bin/bash

set -ex

source "$(dirname "$0")/default.sh"

_base_url="https://github.com/kubevirt/virt-template/releases/download"
_yaml_path="pkg/virt-operator/resource/generate/components/data/virt-template/install-virt-operator.yaml"

curl \
    -L "${_base_url}/${virt_template_version}/install-virt-operator.yaml" \
    -o "${_yaml_path}"
echo "${virt_template_yaml_sha256} ${_yaml_path}" | sha256sum --check --strict
