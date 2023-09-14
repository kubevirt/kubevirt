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
source hack/bootstrap.sh
source hack/config.sh

bazel run \
    --config=${HOST_ARCHITECTURE} \
    //vendor/mvdan.cc/sh/v3/cmd/shfmt:shfmt -- -i 4 -w ${KUBEVIRT_DIR}/hack/ ${KUBEVIRT_DIR}/images/
bazel run \
    --config=${HOST_ARCHITECTURE} \
    //:gazelle -- pkg/ tools/ tests/ cmd/
bazel run \
    --config=${HOST_ARCHITECTURE} \
    //:goimports
# align BAZEL files to a single format
bazel run \
    --config=${HOST_ARCHITECTURE} \
    //:buildifier
