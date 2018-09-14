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

set -eo pipefail

script_dir="$(readlink -f $(dirname $0))"
source "${script_dir}"/common.sh
source "${script_dir}"/config.sh

targets="${@:-${DOCKER_IMAGES}}"

for tgt in ${targets}; do
    bin_name="$(basename ${tgt})"
    bin_path="${tgt%/}"
    dest_dir="${OUT_DIR}/${bin_path}/"
    echo "$dest_dir"
    # Cloner has no build artifact, copy cloner_startup.sh as well
    if [[ "${bin_name}" == "${CLONER}" ]]; then
        mkdir -p "${CMD_OUT_DIR}/${bin_name}/"
        cp -f "${CDI_DIR}/cmd/${bin_name}/cloner_startup.sh" "${dest_dir}"
    fi
    # Copy respective docker files to the directory of the build artifact
    cp -f "${BUILD_DIR}/docker/${bin_name}/Dockerfile" "${dest_dir}"
done
