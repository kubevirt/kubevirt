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
# Copyright 2017 Red Hat, Inc.
#

set -ex

source hack/common.sh
source hack/config.sh

# Bazel runs in the builder container. The binary runs on the host, where the
# cluster kubeconfig is available. Set ARTIFACTS only after this call:
# hack/dockerized rejects a relative ARTIFACTS path.
hack/dockerized 'source hack/common.sh &&
    source hack/bootstrap.sh &&
    source hack/config.sh &&
    mkdir -p "${CMD_OUT_DIR}/dump" &&
    bazel build \
        --config="${HOST_ARCHITECTURE}" "${BAZEL_CS_CONFIG}" \
        //cmd/dump:dump &&
    bazel run \
        --config="${HOST_ARCHITECTURE}" "${BAZEL_CS_CONFIG}" \
        :build-dump -- "${CMD_OUT_DIR}/dump/dump"'

export ARTIFACTS=${ARTIFACTS:-_out/artifacts}

${CMD_OUT_DIR}/dump/dump --kubeconfig=${kubeconfig}
