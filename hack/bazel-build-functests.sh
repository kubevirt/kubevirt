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

set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

# Add the forbid-focus-container ginkgolinter flag when running in CI
# to avoid focus containers sneaking past and being accidentally committed
if [ -n "${CI}" ]; then
    # jq still doesn't support in place editing of files
    jq '.ginkgolinter.analyzer_flags |= . + {"forbid-focus-container":"true"}' nogo_config.json >nogo_config_ci.json && mv nogo_config_ci.json nogo_config.json
fi

rm -rf "${TESTS_OUT_DIR}"
mkdir -p "${TESTS_OUT_DIR}/tools"
mkdir -p "${CMD_OUT_DIR}/dump"
mkdir -p "${CMD_OUT_DIR}/virtctl"
mkdir -p "${CMD_OUT_DIR}/example-guest-agent"

# The --remote_download_toplevel command does not work with run targets out of
# the box. In order to make run targets work with reduced artifact downloads,
# we have to explicitly call them as top-level targets, so that they get
# downloaded.
bazel build \
    --config=${HOST_ARCHITECTURE} \
    //cmd/virtctl:virtctl \
    //cmd/dump:dump \
    //tools/manifest-templator:templator \
    //vendor/github.com/onsi/ginkgo/v2/ginkgo:ginkgo \
    //tests:go_default_test \
    //tools/junit-merger:junit-merger \
    //cmd/example-guest-agent:example-guest-agent

bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-ginkgo -- ${TESTS_OUT_DIR}/ginkgo
bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-functests -- ${TESTS_OUT_DIR}/tests.test
bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-junit-merger -- ${TESTS_OUT_DIR}/junit-merger
bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-dump -- ${CMD_OUT_DIR}/dump/dump
bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl
bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-example-guest-agent -- ${CMD_OUT_DIR}/example-guest-agent/example-guest-agent
