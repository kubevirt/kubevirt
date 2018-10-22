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
        -controller-image="${CONTROLLER_IMAGE_NAME}" \
        -importer-image="${IMPORTER_IMAGE_NAME}" \
        -cloner-image="${CLONER_IMAGE_NAME}" \
        -apiserver-image=${APISERVER_IMAGE_NAME} \
        -uploadproxy-image=${UPLOADPROXY_IMAGE_NAME} \
        -uploadserver-image=${UPLOADSERVER_IMAGE_NAME} \
        -verbosity="${VERBOSITY}" \
        -pull-policy="${PULL_POLICY}" \
        -namespace="${NAMESPACE}"
    ) 1>"${MANIFEST_GENERATED_DIR}/${outFile}"

    (${generator} -template="${tmpl}" \
        -docker-repo="{{ docker_prefix }}" \
        -docker-tag="{{ docker_tag }}" \
        -controller-image="{{ controller_image }}" \
        -importer-image="{{ importer_image }}" \
        -cloner-image="{{ cloner_image }}" \
        -apiserver-image="{{ apiserver_image }}" \
        -uploadproxy-image="{{ uploadproxy_image }}" \
        -uploadserver-image="{{ uploadserver_image }}" \
        -verbosity="{{ verbosity }}" \
        -pull-policy="{{ pull_policy }}" \
        -namespace="{{ cdi_namespace }}"
    ) 1>"${MANIFEST_GENERATED_DIR}/${outFile}.j2"
done

# Remove empty lines at the end of files which are added by go templating
find ${MANIFEST_GENERATED_DIR}/ -type f -exec sed -i {} -e '${/^$/d;}' \;
