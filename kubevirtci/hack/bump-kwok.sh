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
# Copyright The KubeVirt Authors.

set -e

KWOK_RELEASES="https://github.com/kubernetes-sigs/kwok/releases/download/"

# syntax:
# ./hack/bump-kwok.sh <PROVIDER> <KWOK_VERSION>

# usage example
# ./hack/bump-kwok.sh 1.28 v0.5.2

function main() {
    provider="${1:?provider not set or empty}"
    kwok_version="${2:?kwok version not set or empty}"

    declare -a manifests_url
    manifests_url+=("${KWOK_RELEASES}/${kwok_version}/kwok.yaml")
    manifests_url+=("${KWOK_RELEASES}/${kwok_version}/stage-fast.yaml")


    declare -a manifests
    for url in "${manifests_url[@]}"; do
        file="${url##*/}"
        if ! ls "./cluster-provision/k8s/${provider}/manifests/kwok/${file}" > /dev/null; then
            echo "${file} not found at kubevirtci folder"
            exit 1
        fi

        manifest=$(curl -Ls "${url}")
        if [[ "${manifest}" == "Not Found" ]]; then
            echo "${url} not found"
            exit 1
        fi

        manifests+=("${manifest}")
    done

    for i in "${!manifests[@]}"; do
        file="${manifests_url[i]##*/}"
        echo "${manifests[$i]}" > "./cluster-provision/k8s/${provider}/manifests/kwok/${file}"
    done

    echo "kwok, provision, Bump k8s-${provider} kwok to ${kwok_version}"
}

main "$@"
