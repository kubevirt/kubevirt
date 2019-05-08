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
source hack/config.sh

bazel build \
    --sandbox_debug \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_ppc64le \
    --workspace_status_command=./hack/print-workspace-status.sh \
    --define container_prefix=${docker_prefix} \
    --define container_tag=${docker_tag} \
    //:build-images

if [ -n "$docker_tag_alt" ]; then
    bazel build \
        --platforms=@io_bazel_rules_go//go/toolchain:linux_ppc64le \
        --workspace_status_command=./hack/print-workspace-status.sh \
        --define container_prefix=${docker_prefix} \
        --define container_tag=${tag} \
        //:build-images
done
