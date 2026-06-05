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
# Copyright the KubeVirt Authors.
#
#

set -ex

source "$(dirname "$0")/common.sh"
source "$(dirname "$0")/config.sh"

function getStableProvider() {
    new_kubevirtci_git_hash="$1"
    for k8s_provider in $(cd kubevirtci/cluster-up/cluster && ls -rd k8s-[0-9]\.[0-9][0-9]); do
        # shellcheck disable=SC2154
        k8s_provider_version=$(curl --fail "https://raw.githubusercontent.com/kubevirt/kubevirtci/${new_kubevirtci_git_hash}/cluster-provision/k8s/${k8s_provider#"k8s-"}/version")
        if [[ ! "${k8s_provider_version}" =~ -(rc|alpha|beta) ]]; then
            echo "${k8s_provider}"
            return
        fi
    done
    echo "No stable provider found"
    exit 1
}

function main() {
    echo $(getStableProvider "${kubevirtci_git_hash}") >./kubevirtci/stable_provider.txt
}

main "$@"
