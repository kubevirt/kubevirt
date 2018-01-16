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

PROVIDER=${PROVIDER:-vagrant-kubernetes}

source hack/common.sh
source cluster/$PROVIDER/provider.sh
source hack/config.sh

echo "Cleaning up ..."
# Work around https://github.com/kubernetes/kubernetes/issues/33517
_kubectl delete ds -l "kubevirt.io" -n kube-system --cascade=false --grace-period 0 2>/dev/null || :
_kubectl delete pods -n kube-system -l="kubevirt.io=libvirt" --force --grace-period 0 2>/dev/null || :
_kubectl delete pods -n kube-system -l="kubevirt.io=virt-handler" --force --grace-period 0 2>/dev/null || :

# Delete everything, no matter if release, devel or infra
_kubectl delete -f ${MANIFESTS_OUT_DIR}/ -R --grace-period 1 2>/dev/null || :

# Delete exposures
_kubectl delete services -l "kubevirt.io" -n kube-system

sleep 2

echo "Deploying ..."

# Deploy the right manifests for the right target
if [ -z "$TARGET" ] || [ "$TARGET" = "vagrant-dev" ]; then
    _kubectl create -f ${MANIFESTS_OUT_DIR}/dev -R $i
elif [ "$TARGET" = "vagrant-release" ]; then
    _kubectl create -f ${MANIFESTS_OUT_DIR}/release -R $i
fi

# Deploy additional infra for testing
_kubectl create -f ${MANIFESTS_OUT_DIR}/testing -R $i

if [ "$PROVIDER" = "vagrant-openshift" ]; then
    _kubectl adm policy add-scc-to-user privileged -z kubevirt-infra -n kube-system
fi

echo "Done"
