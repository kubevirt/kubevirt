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
# Copyright 2018 Red Hat, Inc.
#

set -ex

DOCKER_TAG=${DOCKER_TAG:-devel}

source hack/common.sh
source cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

function dump_kubevirt() {
    echo "Dump kubevirt state:"
    _kubectl get all -n kubevirt
    for operator in $(kubectl -n kubevirt get pods | grep operator | awk '{print $1}'); do
        echo "Logs for operator $operator"
        _kubectl logs -n kubevirt $operator
    done
}

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

if [[ "$KUBEVIRT_PROVIDER" =~ os-* ]] || [[ "$KUBEVIRT_PROVIDER" =~ okd-* ]]; then
    # Helpful for development. Allows admin to access everything KubeVirt creates in the web console
    _kubectl adm policy add-scc-to-user privileged admin
fi

if [[ "$KUBEVIRT_PROVIDER" =~ kind.* ]]; then
    #removing it since it's crashing with dind because loopback devices are shared with the host
    _kubectl delete -n kubevirt ds disks-images-provider
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
_kubectl wait -n kubevirt kv kubevirt --for condition=Available --timeout 6m || (echo "KubeVirt not ready in time" && dump_kubevirt && exit 1)

echo "Done"
