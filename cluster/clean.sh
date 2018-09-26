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

# Remove finalizers from all running vmis, to not block the cleanup
cluster/kubectl.sh get vmis --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep foregroundDeleteVirtualMachine | while read p; do
    arr=($p)
    name="${arr[0]}"
    namespace="${arr[1]}"
    _kubectl patch vmi $name -n $namespace --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'
done

# Work around https://github.com/kubernetes/kubernetes/issues/33517
_kubectl delete ds -l "kubevirt.io" -n ${namespace} --cascade=false --grace-period 0 2>/dev/null || :
_kubectl delete pods -n ${namespace} -l="kubevirt.io=libvirt" --force --grace-period 0 2>/dev/null || :
_kubectl delete pods -n ${namespace} -l="kubevirt.io=virt-handler" --force --grace-period 0 2>/dev/null || :

# Delete all traces of kubevirt
namespaces=(default ${namespace})
for i in ${namespaces[@]}; do
    _kubectl -n ${i} delete apiservices -l 'kubevirt.io'
    _kubectl -n ${i} delete deployment -l 'kubevirt.io'
    _kubectl -n ${i} delete rs -l 'kubevirt.io'
    _kubectl -n ${i} delete services -l 'kubevirt.io'
    _kubectl -n ${i} delete apiservices -l 'kubevirt.io'
    _kubectl -n ${i} delete validatingwebhookconfiguration -l 'kubevirt.io'
    _kubectl -n ${i} delete secrets -l 'kubevirt.io'
    _kubectl -n ${i} delete pvc -l 'kubevirt.io'
    _kubectl -n ${i} delete pv -l 'kubevirt.io'
    _kubectl -n ${i} delete ds -l 'kubevirt.io'
    _kubectl -n ${i} delete customresourcedefinitions -l 'kubevirt.io'
    _kubectl -n ${i} delete pods -l 'kubevirt.io'
    _kubectl -n ${i} delete clusterrolebinding -l 'kubevirt.io'
    _kubectl -n ${i} delete rolebinding -l 'kubevirt.io'
    _kubectl -n ${i} delete roles -l 'kubevirt.io'
    _kubectl -n ${i} delete clusterroles -l 'kubevirt.io'
    _kubectl -n ${i} delete serviceaccounts -l 'kubevirt.io'
    _kubectl -n ${i} delete configmaps -l 'kubevirt.io'
done

# delete all traces of CDI
for i in ${namespaces[@]}; do
    _kubectl -n ${i} delete deployment -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete services -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete apiservices -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete validatingwebhookconfiguration -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete secrets -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete pvc -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete customresourcedefinitions -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete pods -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete clusterrolebinding -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete rolebinding -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete roles -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete clusterroles -l 'cdi.kubevirt.io'
    _kubectl -n ${i} delete serviceaccounts -l 'cdi.kubevirt.io'
done

sleep 2

echo "Done"
