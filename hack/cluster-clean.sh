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

set -ex

DOCKER_TAG=${DOCKER_TAG:-devel}

source hack/common.sh
source kubevirtci/cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

function patch_remove_finalizers() {
    _kubectl patch --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]' $@
}

function delete_kubevirt_cr() {
    # Delete KubeVirt CR, timeout after 10 seconds
    set +e
    _kubectl -n ${namespace} delete kv kubevirt --timeout=10s --ignore-not-found || true
    _kubectl -n cdi delete service cdi-uploadproxy-nodeport || true
    patch_remove_finalizers -n ${namespace} kv kubevirt
    set -e
}

function remove_finalizers() {
    _kubectl get vmsnapshots --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep vmsnapshot-protection | while read p; do
        local arr=($p)
        local name="${arr[0]}"
        local ns="${arr[1]}"
        patch_remove_finalizers -n $ns vmsnapshots $name
    done

    _kubectl get vmrestores --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep vmrestore-protection | while read p; do
        local arr=($p)
        local name="${arr[0]}"
        local ns="${arr[1]}"
        patch_remove_finalizers -n $ns vmrestores $name
    done

    _kubectl get vmsnapshotcontents --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep vmsnapshotcontent-protection | while read p; do
        local arr=($p)
        local name="${arr[0]}"
        local ns="${arr[1]}"
        patch_remove_finalizers -n $ns vmsnapshotcontents $name
    done

    # Remove finalizers from all running vmis, to not block the cleanup
    _kubectl get vmis --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep foregroundDeleteVirtualMachine | while read p; do
        local arr=($p)
        local name="${arr[0]}"
        local ns="${arr[1]}"
        patch_remove_finalizers -n $ns vmi $name
    done

    _kubectl get vms --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep -e foregroundDeleteVirtualMachine -e orphan -e snapshot-source-protection | while read p; do
        local arr=($p)
        local name="${arr[0]}"
        local ns="${arr[1]}"
        patch_remove_finalizers -n $ns vm $name
    done
}

function delete_resources() {
    local managed_namespaces=("$@")

    # Delete all traces of kubevirt
    local namespaces=(default ${managed_namespaces[@]})
    local labels=("operator.kubevirt.io" "kubevirt.io")

    # Namespaced resources
    for i in ${namespaces[@]}; do
        for label in ${labels[@]}; do
            _kubectl -n ${i} delete deployment,ds,rs,pods,services,pvc,rolebinding,role,serviceaccounts,configmaps,secrets,jobs -l ${label}
        done
    done

    # Not namespaced resources
    for label in ${labels[@]}; do
        # Remove the finalizers added by virt-operator from CRDs
        _kubectl get customresourcedefinitions --no-headers -o=custom-columns=NAME:.metadata.name,FINALIZERS:.metadata.finalizers -l ${label} | grep -e "kubevirt.io/virtOperatorFinalizer" | while read p; do
            local arr=($p)
            local name="${arr[0]}"
            patch_remove_finalizers customresourcedefinitions ${name}
        done
        _kubectl delete apiservices,clusterroles,clusterrolebinding,customresourcedefinitions,pv,validatingwebhookconfiguration -l ${label}
    done

    _kubectl delete priorityclass kubevirt-cluster-critical --ignore-not-found
}

function delete_namespaces() {
    local managed_namespaces=("$@")

    _kubectl delete ns ${managed_namespaces[@]} --timeout=180s --ignore-not-found
}

function main() {
    echo "Cleaning up ..."

    local kubevirt_managed_namespaces=(${namespace})

    delete_kubevirt_cr
    remove_finalizers
    delete_resources "${kubevirt_managed_namespaces[@]}"
    delete_namespaces "${kubevirt_managed_namespaces[@]}"

    sleep 2

    echo "Done $0"
}

main "$@"
