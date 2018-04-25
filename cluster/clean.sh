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
source cluster/$PROVIDER/provider.sh
source hack/config.sh

echo "Cleaning up ..."

# Remove finalizers from all running vms, to not block the cleanup
cluster/kubectl.sh get vms --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,FINALIZERS:.metadata.finalizers --no-headers | grep foregroundDeleteVirtualMachine | while read p; do
    arr=($p)
    name="${arr[0]}"
    namespace="${arr[1]}"
    _kubectl patch vm $name -n $namespace --type=json -p '[{ "op": "remove", "path": "/metadata/finalizers" }]'
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
    _kubectl -n ${i} delete pv -l 'kubevirt.io'
    _kubectl -n ${i} delete pvc -l 'kubevirt.io'
    _kubectl -n ${i} delete ds -l 'kubevirt.io'
    _kubectl -n ${i} delete customresourcedefinitions -l 'kubevirt.io'
    _kubectl -n ${i} delete pods -l 'kubevirt.io'
    _kubectl -n ${i} delete clusterrolebinding -l 'kubevirt.io'
    _kubectl -n ${i} delete rolebinding -l 'kubevirt.io'
    _kubectl -n ${i} delete roles -l 'kubevirt.io'
    _kubectl -n ${i} delete clusterroles -l 'kubevirt.io'
    _kubectl -n ${i} delete serviceaccounts -l 'kubevirt.io'
    # FIXME this is workaroung to make CI happy. Can be removed in few days.
    if [ $(_kubectl -n ${i} get crd offlinevirtualmachines.kubevirt.io | wc -l) -gt 0 ]; then
        _kubectl -n ${i} delete crd 'offlinevirtualmachines.kubevirt.io'
    fi
done

sleep 2

echo "Done"
