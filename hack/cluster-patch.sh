#!/bin/bash -xe
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


source hack/common.sh
source kubevirtci/cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

function wait_for_rollout() {
    _kubectl rollout status $1 -n $namespace $2 --timeout=240s
}

function wait_for_digest() {
    for digest in $3; do
        while ! _kubectl get -n $namespace $1 $2 -o yaml | grep $digest; do
            sleep 5
        done
    done
}

function wait_for() {
    wait_for_digest $1 $2 "$3"
    wait_for_rollout $1 $2
}

source ./hack/parse-shasums.sh

_kubectl set env deployment -n $namespace virt-operator \
    GS_SHASUM=$GS_SHA \
    PR_HELPER_SHASUM=$PR_HELPER_SHASUM \
    SIDECAR_SHIM_SHASUM=$SIDECAR_SHIM_SHASUM \
    VIRT_HANDLER_SHASUM=$VIRT_HANDLER_SHA \
    VIRT_LAUNCHER_SHASUM=$VIRT_LAUNCHER_SHA \
    VIRT_CONTROLLER_SHASUM=$VIRT_CONTROLLER_SHA \
    VIRT_API_SHASUM=$VIRT_API_SHA \
    VIRT_EXPORTPROXY_SHASUM=$VIRT_EXPORTPROXY_SHA \
    VIRT_EXPORTSERVER_SHASUM=$VIRT_EXPORTSERVER_SHA

wait_for ds virt-handler "$VIRT_LAUNCHER_SHA $VIRT_HANDLER_SHA"
wait_for deployment virt-controller "$VIRT_LAUNCHER_SHA $VIRT_CONTROLLER_SHA"
wait_for deployment virt-api $VIRT_API_SHA
# TODO XXX this may have to be optional
# wait_for deployment virt-exportproxy $VIRT_EXPORTPROXY_SHA
