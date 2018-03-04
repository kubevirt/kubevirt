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
    _kubectl adm policy add-scc-to-user privileged -z kubevirt-controller -n kube-system
    _kubectl adm policy add-scc-to-user privileged -z kubevirt-testing -n kube-system
    _kubectl adm policy add-scc-to-user privileged -z kubevirt-privileged -n kube-system
fi

echo "Done"
