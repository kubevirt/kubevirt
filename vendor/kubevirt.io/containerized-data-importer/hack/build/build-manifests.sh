#!/usr/bin/env bash

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

templates="$(find "${MANIFEST_TEMPLATE_DIR}" -name *.in -type f)"
generator="${BIN_DIR}/manifest-generator"

(cd "${CDI_DIR}/tools/manifest-generator/" && go build -o "${generator}" ./...)

mkdir -p "${MANIFEST_GENERATED_DIR}/"

for tmpl in ${templates}; do
    tmpl=$(readlink -f "${tmpl}")
    outFile=$(basename -s .in "${tmpl}")
    rm -rf "${MANIFEST_GENERATED_DIR}/${outFile}"
    (${generator} -template="${tmpl}" \
        -docker-repo="${DOCKER_REPO}" \
        -docker-tag="${DOCKER_TAG}" \
        -verbosity="${VERBOSITY}" \
        -pull-policy="${PULL_POLICY}"
    ) 1>"${MANIFEST_GENERATED_DIR}/${outFile}"
done
