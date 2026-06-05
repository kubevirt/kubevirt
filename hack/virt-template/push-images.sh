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
# Copyright The KubeVirt Authors.
#

set -e

source hack/common.sh
source hack/config.sh
source hack/virt-template/default.sh

virt_template_targets="virt-template-apiserver virt-template-controller"

for target in ${virt_template_targets}; do
    bazel run \
        --config="${ARCHITECTURE}" \
        "//:push-${target}" -- --repository "${docker_prefix}/${image_prefix}${target}" --tag "${virt_template_version}"
done

# for the imagePrefix operator test
if [[ $image_prefix_alt ]]; then
    for target in ${virt_template_targets}; do
        bazel run \
            --config="${ARCHITECTURE}" \
            "//:push-${target}" -- --repository "${docker_prefix}/${image_prefix_alt}${target}" --tag "${virt_template_version}"
    done
fi
