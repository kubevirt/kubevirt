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

if [[ ! -f "${KUBEVIRT_DIR}/bazel-bin/push-virt-operator.digest" ]]; then
    echo "digest files not found: won't use shasums, falling back to tags"
    return
fi

if [[ ${KUBEVIRT_ONLY_USE_TAGS} == "true" ]]; then
    echo "found KUBEVIRT_ONLY_USE_TAGS; using tags instead of shasums"
    return
fi

# bazel push images creates digest files in bazel-bin/push-<image-name>.digest

VIRT_OPERATOR_SHA=$(cat ${KUBEVIRT_DIR}/bazel-bin/push-virt-operator.digest)
VIRT_API_SHA=$(cat ${KUBEVIRT_DIR}/bazel-bin/push-virt-api.digest)
VIRT_CONTROLLER_SHA=$(cat ${KUBEVIRT_DIR}/bazel-bin/push-virt-controller.digest)
VIRT_HANDLER_SHA=$(cat ${KUBEVIRT_DIR}/bazel-bin/push-virt-handler.digest)
VIRT_LAUNCHER_SHA=$(cat ${KUBEVIRT_DIR}/bazel-bin/push-virt-launcher.digest)
