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
#

set -e
set -x

source hack/config-kubevirtci.sh

TEMP_FILE=$(mktemp -p /tmp -t kubevirt.deploy.XXXX)

trap 'rm -f $TEMP_FILE' EXIT SIGINT

function main() {
    ./hack/cluster-clean.sh >$TEMP_FILE 2>&1 &
    CLEAN_PID=$!

    ./hack/cluster-build.sh
    ./hack/manifests.sh

    echo "waiting for cluster-clean to finish"
    if ! wait $CLEAN_PID; then
        echo "cluster-clean failed, output was:"
        cat $TEMP_FILE
        exit 1
    fi

    ./hack/deploy-to-nodes.sh
    ./hack/cluster-deploy.sh
}

main "$@"
