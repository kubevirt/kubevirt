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
source hack/config.sh

rm -rf "${TESTS_OUT_DIR}"
mkdir -p "${TESTS_OUT_DIR}/tools"
mkdir -p "${CMD_OUT_DIR}/dump"
mkdir -p "${CMD_OUT_DIR}/virtctl"

# The --remote_download_toplevel command does not work with run targets out of
# the box. In order to make run targets work with reduced artifact downloads,
# we have to explicitly call them as top-level targets, so that they get
# downloaded.
bazel build \
    --config=${ARCHITECTURE} \
    //cmd/virtctl:virtctl \
    //cmd/dump:dump \
    //tools/manifest-templator:templator \
    //vendor/github.com/onsi/ginkgo/ginkgo:ginkgo \
    //tests:go_default_test \
    //tools/junit-merger:junit-merger

bazel run \
    --config=${ARCHITECTURE} \
    :build-ginkgo -- ${TESTS_OUT_DIR}/ginkgo
bazel run \
    --config=${ARCHITECTURE} \
    :build-functests -- ${TESTS_OUT_DIR}/tests.test
bazel run \
    --config=${ARCHITECTURE} \
    :build-junit-merger -- ${TESTS_OUT_DIR}/junit-merger
bazel run \
    --config=${ARCHITECTURE} \
    :build-dump -- ${CMD_OUT_DIR}/dump/dump
bazel run \
    --config=${ARCHITECTURE} \
    :build-virtctl -- ${CMD_OUT_DIR}/virtctl/virtctl
