#!/usr/bin/env bash
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
# Copyright 2023 NVIDIA CORPORATION
#

source hack/common.sh

COMMAND=$1

build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)

# Only add the tailing $arch when doing a multi-arch build
if [ "$build_count" -gt 1 ]; then
    for arch in ${BUILD_ARCH//,/ }; do
        echo "[INFO] -- working on $arch --"
        arch=$(format_archname $arch)
        tag=$(format_archname $arch tag)
        DOCKER_TAG=$DOCKER_TAG-$tag ARCHITECTURE=$arch hack/bazel-${COMMAND}.sh
    done
else
    arch=$(format_archname ${BUILD_ARCH})
    ARCHITECTURE=${arch} hack/bazel-${COMMAND}.sh
fi
