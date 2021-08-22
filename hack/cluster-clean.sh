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

set -ex

DOCKER_TAG=${DOCKER_TAG:-devel}

source hack/common.sh
source cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

function delete_kubevirt_cr() {
    # Delete KubeVirt CR, timeout after 10 seconds
    set +e
    (
        local cmdpid=$BASHPID
        (
            sleep 10
            kill $cmdpid
        ) &
        _kubectl -n ${namespace} delete kv kubevirt
    )
    _kubectl -n ${namespace} patch kv kubevirt --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'
    _kubectl patch cdi cdi --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'

    set -e
}

function remove_finalizers() {
    kubectl get vmsnapshots --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep vmsnapshot-protection | while read p; do
        local arr=($p)
        local name="${arr[0]}"
        local ns="${arr[1]}"
        _kubectl patch vmsnapshots $name -n $ns --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'
    done

    kubectl get vmsnapshotcontents --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep vmsnapshotcontent-protection | while read p; do
        local arr=($p)
        local name="${arr[0]}"
        local ns="${arr[1]}"
        _kubectl patch vmsnapshotcontents $name -n $ns --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'
    done

    # Remove finalizers from all running vmis, to not block the cleanup
    _kubectl get vmis --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep foregroundDeleteVirtualMachine | while read p; do
        local arr=($p)
        local name="${arr[0]}"
        local ns="${arr[1]}"
        _kubectl patch vmi $name -n $ns --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'
    done

    _kubectl get vms --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep -e foregroundDeleteVirtualMachine -e orphan -e snapshot-source-protection | while read p; do
        local arr=($p)
        local name="${arr[0]}"
        local ns="${arr[1]}"
        _kubectl patch vm $name -n $ns --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'
    done
}

function delete_resources() {
    local managed_namespaces=("$@")

    # Delete all traces of kubevirt
    local namespaces=(default ${managed_namespaces[@]})
    local labels=("operator.kubevirt.io" "operator.cdi.kubevirt.io" "kubevirt.io" "cdi.kubevirt.io")

    # Namespaced resources
    for i in ${namespaces[@]}; do
        for label in ${labels[@]}; do
            _kubectl -n ${i} delete deployment,ds,rs,pods,services,pvc,rolebinding,role,serviceaccounts,configmaps,secrets,jobs -l ${label}
        done
    done

    # Not namespaced resources
    for label in ${labels[@]}; do
        _kubectl delete apiservices,clusterroles,clusterrolebinding,customresourcedefinitions,pv,validatingwebhookconfiguration -l ${label}
    done
}

function delete_namespaces() {
    local managed_namespaces=("$@")

    _kubectl delete ns ${managed_namespaces[@]} --timeout=180s --ignore-not-found
}

function main() {
    echo "Cleaning up ..."

    local kubevirt_managed_namespaces=(${namespace} ${cdi_namespace})

    delete_kubevirt_cr
    remove_finalizers
    delete_resources "${kubevirt_managed_namespaces[@]}"
    delete_namespaces "${kubevirt_managed_namespaces[@]}"

    sleep 2

    echo "Done $0"
}

main "$@"
