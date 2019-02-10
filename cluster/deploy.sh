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
# Copyright 2018 Red Hat, Inc.
#

set -ex

source hack/common.sh
source cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

echo "Deploying ..."

# Create the installation namespace if it does not exist already
_kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Namespace
metadata:
  name: ${namespace}
EOF

# Deploy infra for testing first
_kubectl create -f ${MANIFESTS_OUT_DIR}/testing

# Deploy CDI with operator.
_kubectl apply -f - <<EOF
---
apiVersion: cdi.kubevirt.io/v1alpha1
kind: CDI
metadata:
  name: cdi
EOF

# Deploy kubevirt operator
_kubectl apply -f ${MANIFESTS_OUT_DIR}/release/kubevirt-operator.yaml

if [[ "$KUBEVIRT_PROVIDER" =~ os-* ]]; then
    _kubectl create -f ${MANIFESTS_OUT_DIR}/testing/ocp

    _kubectl adm policy add-scc-to-user privileged -z kubevirt-operator -n ${namespace}

    # Helpful for development. Allows admin to access everything KubeVirt creates in the web console
    _kubectl adm policy add-scc-to-user privileged admin
fi

# Ensure the KubeVirt CRD is created
count=0
until _kubectl get crd kubevirts.kubevirt.io; do
    ((count++)) && ((count == 30)) && echo "KubeVirt CRD not found" && exit 1
    echo "waiting for KubeVirt CRD"
    sleep 1
done

# Deploy KubeVirt
_kubectl create -n ${namespace} -f ${MANIFESTS_OUT_DIR}/release/kubevirt-cr.yaml

# Ensure the KubeVirt CR is created
count=0
until _kubectl -n kubevirt get kv kubevirt; do
    ((count++)) && ((count == 30)) && echo "KubeVirt CR not found" && exit 1
    echo "waiting for KubeVirt CR"
    sleep 1
done

# wait until KubeVirt is ready
_kubectl wait -n kubevirt kv kubevirt --for condition=Ready --timeout 180s || (echo "KubeVirt not ready in time" && exit 1)

echo "Done"
