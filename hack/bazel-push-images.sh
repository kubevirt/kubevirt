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
# Copyright 2019 Red Hat, Inc.
#

set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

PUSH_TARGETS=(${PUSH_TARGETS:-other-images virt-operator virt-api virt-controller virt-handler virt-launcher conformance libguestfs})

for tag in ${docker_tag} ${docker_tag_alt}; do
    for target in ${PUSH_TARGETS[@]}; do

        bazel run \
            --config=${ARCHITECTURE} \
            --define container_prefix=${docker_prefix} \
            --define image_prefix=${image_prefix} \
            --define container_tag=${tag} \
            //:push-${target}

    done
done

# for the imagePrefix operator test
if [[ $image_prefix_alt ]]; then
    for target in ${PUSH_TARGETS[@]}; do

        bazel run \
            --config=${ARCHITECTURE} \
            --define container_prefix=${docker_prefix} \
            --define image_prefix=${image_prefix_alt} \
            --define container_tag=${docker_tag} \
            //:push-${target}

    done
fi

rm -rf ${DIGESTS_DIR}
mkdir -p ${DIGESTS_DIR}

for f in $(find bazel-bin/ -name '*.digest'); do
    dir=${DIGESTS_DIR}/$(dirname $f)
    mkdir -p ${dir}
    cp -f ${f} ${dir}/$(basename ${f})
done
