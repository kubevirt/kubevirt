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
    //tools/csv-generator/... //cmd/... //staging/src/kubevirt.io/client-go/examples/...

# Copy dump binary to a reachable place outside of the build container
bazel run \
    --config=${ARCHITECTURE} \
    :build-dump -- ${CMD_OUT_DIR}/dump/dump

# build platform native virtctl explicitly
bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl

# cross-compile virtctl for

# linux
bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-virtctl-amd64 -- ${CMD_OUT_DIR}/virtctl/virtctl-${KUBEVIRT_VERSION}-linux-amd64

# darwin
bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-virtctl-darwin -- ${CMD_OUT_DIR}/virtctl/virtctl-${KUBEVIRT_VERSION}-darwin-amd64

# windows
bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-virtctl-windows -- ${CMD_OUT_DIR}/virtctl/virtctl-${KUBEVIRT_VERSION}-windows-amd64.exe
