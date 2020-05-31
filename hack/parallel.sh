#!/bin/bash -e
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License").
# You may not use this file except in compliance with the License.
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
# Copyright 2020 Red Hat, Inc.
#

# This script is meant for faster cluster-sync.
# It checks if the kubevirt namespace exists, skipping cleaning of the cluster if it doesn't.
# If the ns does exists, it would run the clean parallel to the build, and once both
# are finished it will deploy the cluster.

NAMESPACE=${KUBEVIRT_INSTALLED_NAMESPACE:-kubevirt}
TEMP_FILE=$(mktemp -p /tmp -t kubevirt.deploy.XXXX)

trap 'rm -f $TEMP_FILE' EXIT SIGINT SIGTERM

function _kubectl() {
    cluster-up/kubectl.sh "$@"
}

function clean_and_build() {
    echo "Kubevirt namespace found, cleaning"
    ./hack/cluster-clean.sh >$TEMP_FILE 2>&1 &
    CLEAN_PID=$!

    ./hack/cluster-build.sh

    wait $CLEAN_PID
    if [ $? -ne 0 ]; then
        echo "Clean failed, output was:"
        cat $TEMP_FILE
        exit 1
    fi
}

function build() {
    echo "Kubevirt namespace not found, skipping clean"
    ./hack/cluster-build.sh
}

function main() {
    if ! _kubectl get nodes >/dev/null 2>&1; then
        echo "Cluster not found, exiting"
        exit 1
    fi

    _kubectl get ns $NAMESPACE >/dev/null 2>&1 && clean_and_build || build
    ./hack/cluster-deploy.sh
}

main "$@"
