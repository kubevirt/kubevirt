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
mkdir -p ${CMD_OUT_DIR}/dump

# Build all binaries for amd64
bazel build \
    --config=${ARCHITECTURE} \
    --stamp \
    //tools/csv-generator/... //cmd/... //staging/src/kubevirt.io/client-go/examples/...

# Copy dump binary to a reachable place outside of the build container
bazel run \
    --stamp \
    :build-dump -- ${CMD_OUT_DIR}/dump/dump

# build platform native virtctl explicitly
bazel run \
    --stamp \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl

# cross-compile virtctl for

# linux
bazel run \
    --config=${ARCHITECTURE} \
    --stamp \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl-${KUBEVIRT_VERSION}-linux-${ARCHITECTURE}

# darwin
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:darwin_amd64 \
    --stamp \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl-${KUBEVIRT_VERSION}-darwin-amd64

# windows
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:windows_amd64 \
    --stamp \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl-${KUBEVIRT_VERSION}-windows-amd64.exe
