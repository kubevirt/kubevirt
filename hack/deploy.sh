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
# Copyright 2017 Red Hat, Inc.
#

source hack/common.sh

# create namespaces
"${CMD}" create ns kubevirt-hyperconverged

if [ "${CMD}" == "oc" ]; then
    # Switch project to kubevirt
    oc project kubevirt-hyperconverged
else
    # switch namespace to kubevirt
    ${CMD} config set-context $(${CMD} config current-context) --namespace=kubevirt-hyperconverged
fi

CONTAINER_ERRORED=""
function debug(){
    echo "Found pods with errors ${CONTAINER_ERRORED}"

    for err in "${CONTAINER_ERRORED}"; do
	echo "------------- $err"
	"${CMD}" logs $("${CMD}" get pods -n kubevirt-hyperconverged | grep $err | head -1 | awk '{ print $1 }')
    done
    exit 1
}

# Deploy local manifests
"${CMD}" create -f deploy/cluster_role.yaml
"${CMD}" create -f deploy/service_account.yaml
"${CMD}" create -f deploy/cluster_role_binding.yaml
"${CMD}" create -f deploy/crds/
"${CMD}" create -f deploy/operator.yaml

# Wait for the HCO to be ready
sleep 20

"${CMD}" wait deployment/hyperconverged-cluster-operator --for=condition=Available --timeout="360s" || CONTAINER_ERRORED+="${op}"

for op in cdi-operator cluster-network-addons-operator kubevirt-ssp-operator node-maintenance-operator virt-operator; do
    "${CMD}" wait deployment/"${op}" --for=condition=Available --timeout="360s" || CONTAINER_ERRORED+="${op}"
done

"${CMD}" create -f deploy/hco.cr.yaml
sleep 30
"${CMD}" wait pod $("${CMD}" get pods | grep hyperconverged-cluster-operator | awk '{ print $1 }') --for=condition=Ready --timeout="360s"

for dep in cdi-apiserver cdi-deployment cdi-uploadproxy virt-api virt-controller; do
    "${CMD}" wait deployment/"${dep}" --for=condition=Available --timeout="360s" || CONTAINER_ERRORED+="${dep}"
done

if [ -z "$CONTAINER_ERRORED" ]; then
    echo "SUCCESS"
    exit 0
else
    CONTAINER_ERRORED+='hyperconverged-cluster-operator'
    debug
    "${CMD}" get pods -n kubevirt-hyperconverged
fi
