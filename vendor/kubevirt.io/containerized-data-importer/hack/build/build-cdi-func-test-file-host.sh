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

FILE_INIT="cdi-func-test-file-host-init"
FILE_INIT_PATH="tools/${FILE_INIT}"
FILE_HOST="cdi-func-test-file-host-http"
OUT_PATH="${OUT_DIR}/tools"

mkdir -p "${OUT_PATH}/${FILE_INIT}" "${OUT_PATH}/${FILE_HOST}"

DOCKER_REPO=""

${BUILD_DIR}/build-copy-artifacts.sh "${FILE_INIT_PATH}"

cp ${BUILD_DIR}/docker/${FILE_HOST}/* ${OUT_PATH}/${FILE_HOST}/
cp "${CDI_DIR}/tests/images/tinyCore.iso" "${OUT_PATH}/${FILE_INIT}/"
