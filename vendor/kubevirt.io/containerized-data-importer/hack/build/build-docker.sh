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

set -e

script_dir="$(readlink -f $(dirname $0))"
source "${script_dir}"/common.sh
source "${script_dir}"/config.sh

docker_opt="${1:-build}"
shift
targets="${@:-${DOCKER_IMAGES}}"

printf "Building targets: %s\n" "${targets}"

for tgt in ${targets}; do
    BIN_NAME="$(basename ${tgt})"
    BIN_PATH="${tgt%/}"
    IMAGE="${DOCKER_REPO}/${BIN_NAME}:${DOCKER_TAG}"
    if [ "${docker_opt}" == "build" ]; then
        (
            cd "${OUT_DIR}/${BIN_PATH}"
            docker "${docker_opt}" -t ${IMAGE} .
        )
    elif [ "${docker_opt}" == "push" ]; then
        if [ "${DOCKER_REPO}" == "kubevirt" ]; then
            echo "Pushes to docker.io/kubevirt should only be performed by CI."
            echo "Set DOCKER_REPO and DOCKER_TAG (default :latest) to target other repositories."
            exit 1
        fi
        docker "${docker_opt}" "${IMAGE}"
    elif [ "${docker_opt}" == "publish" ]; then
        if [ -z "${TRAVIS}" ]; then
            echo "Publishing releases should only be performed by the CI. "
            exit 1
        fi
        docker push ${IMAGE}
    fi
done
