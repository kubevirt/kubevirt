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
# Copyright 2021 Red Hat, Inc.
#

set -e

REPO="https://github.com/kubevirt/kubevirtci.git"

TEMP_DIR=$(mktemp -d /tmp/git-tmp.XXXXXX)
trap 'rm -rf $TEMP_DIR' EXIT SIGINT

function main() {
    CURRENT=$(git diff cluster-up/version.txt | grep -v "\-\-" | grep -v "++" | grep "^+" | cut -d - -f 2)
    PREVIOUS=$(git diff cluster-up/version.txt | grep -v "\-\-" | grep -v "++" | grep "^-" | cut -d - -f 3)

    if [[ -z $CURRENT ]] || [[ -z $PREVIOUS ]]; then
        exit 0
    fi

    cd "$TEMP_DIR"
    git clone --depth=100 -n "$REPO" >/dev/null 2>&1
    cd kubevirtci
    info=$(git log $PREVIOUS..$CURRENT --format="%h %s" | sed 's! (#!](https://github.com/kubevirt/kubevirtci/pull/!g')
    info=$(echo "$info" | sed 's/^/\[/')
    info+='\n\n```release-note\nNONE\n```'
    decorated_info=$(echo -e "Bump kubevirtci\n\n${info}")
    echo "$decorated_info"
}

main "$@"
