#!/usr/bin/env bash

#Copyright 2018 The CDI Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

set -euo pipefail

script_dir="$(readlink -f $(dirname $0))"
source "${script_dir}"/common.sh
source "${script_dir}"/config.sh

REGISTRY_INIT_PATH="tools/${FUNC_TEST_REGISTRY_INIT}"

${BUILD_DIR}/build-copy-artifacts.sh "${REGISTRY_INIT_PATH}"

OUT_PATH="${OUT_DIR}/tools"

mkdir -p   "${OUT_PATH}/${FUNC_TEST_REGISTRY}"
mkdir -p   "${OUT_PATH}/${FUNC_TEST_REGISTRY_POPULATE}"
mkdir -p   "${OUT_PATH}/${FUNC_TEST_REGISTRY_INIT}"

DOCKER_REPO=""

cp ${BUILD_DIR}/docker/${FUNC_TEST_REGISTRY}/* ${OUT_PATH}/${FUNC_TEST_REGISTRY}/
cp ${BUILD_DIR}/docker/${FUNC_TEST_REGISTRY_POPULATE}/* ${OUT_PATH}/${FUNC_TEST_REGISTRY_POPULATE}/
cp -r ${BUILD_DIR}/docker/${FUNC_TEST_REGISTRY_INIT}/* ${OUT_PATH}/${FUNC_TEST_REGISTRY_INIT}/
cp "${CDI_DIR}/tests/images/tinyCore.iso" ${OUT_PATH}/${FUNC_TEST_REGISTRY_INIT}/

