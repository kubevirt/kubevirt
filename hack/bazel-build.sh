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

rm -rf ${CMD_OUT_DIR}
mkdir -p ${CMD_OUT_DIR}/virtctl

# Build all binaries for amd64
bazel build \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
    --workspace_status_command=./hack/print-workspace-status.sh \
    //cmd/...

# build platform native virtctl explicitly
bazel run \
    --workspace_status_command=./hack/print-workspace-status.sh \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl

# cross-compile virtctl for

# linux
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
    --workspace_status_command=./hack/print-workspace-status.sh \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl-${KUBEVIRT_VERSION}-linux-amd64

# darwin
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:darwin_amd64 \
    --workspace_status_command=./hack/print-workspace-status.sh \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl-${KUBEVIRT_VERSION}-darwin-amd64

# windows
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:windows_amd64 \
    --workspace_status_command=./hack/print-workspace-status.sh \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl-${KUBEVIRT_VERSION}-windows-amd64.exe
