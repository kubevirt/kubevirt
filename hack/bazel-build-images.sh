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
source hack/bootstrap.sh
source hack/config.sh

# vars are uninteresting for the build step, they are interesting for the push step only

case ${ARCHITECTURE} in
"s390x" | "crossbuild-s390x")
    other_images="
        //:build-other-images_s390x
    "
    ;;
"aarch64" | "crossbuild-aarch64")
    other_images="
        //:build-other-images_aarch64
        //tests:conformance_image
    "
    ;;
*)
    other_images="
        //:build-other-images_x86_64
        //tests:conformance_image
    "
    ;;
esac

bazel build \
    --config=${ARCHITECTURE} \
    --define container_prefix= \
    --define image_prefix= \
    --define container_tag= \
    //cmd/virt-operator:virt-operator-image //cmd/virt-api:virt-api-image //cmd/virt-controller:virt-controller-image \
    //cmd/virt-handler:virt-handler-image //cmd/virt-launcher:virt-launcher-image //cmd/virt-exportproxy:virt-exportproxy-image \
    //cmd/virt-exportserver:virt-exportserver-image ${other_images[@]}

rm -rf ${DIGESTS_DIR}/${ARCHITECTURE}
mkdir -p ${DIGESTS_DIR}/${ARCHITECTURE}

for f in $(find bazel-bin/ -name '*.digest'); do
    dir=${DIGESTS_DIR}/${ARCHITECTURE}/$(dirname $f)
    mkdir -p ${dir}
    cp -f ${f} ${dir}/$(basename ${f})
done
