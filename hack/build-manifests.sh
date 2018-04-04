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

manifest_docker_prefix=${manifest_docker_prefix-${docker_prefix}}

args=$(cd ${KUBEVIRT_DIR}/manifests && find * -type f -name "*.yaml.in")

rm -rf ${MANIFESTS_OUT_DIR}

(cd ${KUBEVIRT_DIR}/tools/manifest-templator/ && go build)

for arg in $args; do
    final_out_dir=$(dirname ${MANIFESTS_OUT_DIR}/${arg})
    mkdir -p ${final_out_dir}
    manifest=$(basename -s .in ${arg})
    outfile=${final_out_dir}/${manifest}

    ${KUBEVIRT_DIR}/tools/manifest-templator/manifest-templator \
        --namespace=${namespace} \
        --docker-prefix=${manifest_docker_prefix} \
        --docker-tag=${docker_tag} \
        --generated-manifests-dir=${KUBEVIRT_DIR}/manifests/generated/ \
        --input-file=${KUBEVIRT_DIR}/manifests/$arg >${outfile}
done
