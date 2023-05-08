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

COMMAND=$1
# We are formatting the architecture name here to ensure that
# it is consistent with the platform name specified in ../.bazelrc
function format_archname() {
    local local_platform=$(uname -m)
    local platform=$1

    if [ $# -lt 1 ]; then
        echo ${local_platform}
    else
        case ${platform} in
        x86_64 | amd64)
            arch="x86_64"
            echo ${arch}
            ;;
        crossbuild-aarch64 | aarch64 | arm64)
            if [ ${local_platform} != "aarch64" ]; then
                arch="crossbuild-aarch64"
            else
                arch="aarch64"
            fi
            echo ${arch}
            ;;
        *)
            echo "ERROR: invalid Arch, ${platform}, only support x86_64 and aarch64"
            exit 1
            ;;
        esac
    fi
}

build_count=$(echo ${BUILD_ARCH//,/ } | wc -w)

# Only add the tailing $arch when doing a multi-arch build
if [ "$build_count" -gt 1 ]; then
    for arch in ${BUILD_ARCH//,/ }; do
        echo "[INFO] -- working on $arch --"
        arch=$(format_archname $arch)
        DOCKER_TAG=$DOCKER_TAG-$arch ARCHITECTURE=$arch hack/bazel-${COMMAND}.sh
    done
else
    arch=$(format_archname ${BUILD_ARCH})
    ARCHITECTURE=${arch} hack/bazel-${COMMAND}.sh
fi
