#!/bin/bash
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

source hack/common.sh
source cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

echo "Cleaning up ..."

# Delete KubeVirt CR, timeout after 10 seconds
set +e
(
    cmdpid=$BASHPID
    (
        sleep 10
        kill $cmdpid
    ) &
    _kubectl -n ${namespace} delete kv kubevirt
)
_kubectl -n ${namespace} patch kv kubevirt --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'

set -e

# Remove finalizers from all running vmis, to not block the cleanup
_kubectl get vmis --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep foregroundDeleteVirtualMachine | while read p; do
    arr=($p)
    name="${arr[0]}"
    namespace="${arr[1]}"
    _kubectl patch vmi $name -n $namespace --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'
done

# Delete all traces of kubevirt
namespaces=(default ${namespace} ${cdi_namespace})
labels=("operator.kubevirt.io" "kubevirt.io" "cdi.kubevirt.io")

# Namespaced resources
for i in ${namespaces[@]}; do
    for label in ${labels[@]}; do
        _kubectl -n ${i} delete deployment -l ${label}
        _kubectl -n ${i} delete ds -l ${label}
        _kubectl -n ${i} delete rs -l ${label}
        _kubectl -n ${i} delete pods -l ${label}
        _kubectl -n ${i} delete services -l ${label}
        _kubectl -n ${i} delete pvc -l ${label}
        _kubectl -n ${i} delete rolebinding -l ${label}
        _kubectl -n ${i} delete roles -l ${label}
        _kubectl -n ${i} delete serviceaccounts -l ${label}
        _kubectl -n ${i} delete configmaps -l ${label}
        _kubectl -n ${i} delete secrets -l ${label}
    done
done

# Not namespaced resources
for label in ${labels[@]}; do
    _kubectl delete validatingwebhookconfiguration -l ${label}
    _kubectl delete pv -l ${label}
    _kubectl delete clusterrolebinding -l ${label}
    _kubectl delete clusterroles -l ${label}
    _kubectl delete customresourcedefinitions -l ${label}

    if [[ "$KUBEVIRT_PROVIDER" =~ os-* ]]; then
        _kubectl delete scc -l ${label}
    fi

    # W/A for https://github.com/kubernetes/kubernetes/issues/65818
    if [[ "$KUBEVIRT_PROVIDER" =~ .*.10..* ]]; then
        # k8s version 1.10.* does not have --wait parameter
        _kubectl delete apiservices -l ${label}
    else
        _kubectl delete apiservices -l ${label} --wait=false
    fi
    _kubectl get apiservices -l ${label} -o=custom-columns=NAME:.metadata.name,FINALIZERS:.metadata.finalizers --no-headers | grep foregroundDeletion | while read p; do
        arr=($p)
        name="${arr[0]}"
        _kubectl -n ${i} patch apiservices $name --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'
    done
done

if [ -n "$(_kubectl get ns | grep "${namespace} ")" ]; then
    echo "Clean ${namespace} namespace"
    _kubectl delete ns ${namespace}

    start_time=0
    sample=10
    timeout=120
    echo "Waiting for ${namespace} namespace to disappear ..."
    while [ -n "$(_kubectl get ns | grep "${namespace} ")" ]; do
        sleep $sample
        start_time=$((current_time + sample))
        if [[ $current_time -gt $timeout ]]; then
            exit 1
        fi
    done
fi

sleep 2

echo "Done"
