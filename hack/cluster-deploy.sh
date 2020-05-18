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
    hack/dump.sh
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

if [[ "$KUBEVIRT_STORAGE" == "rook-ceph" ]]; then
    _kubectl apply -f ${KUBEVIRT_DIR}/manifests/testing/external-snapshotter
    _kubectl apply -f ${KUBEVIRT_DIR}/manifests/testing/rook-ceph/common.yaml
    _kubectl apply -f ${KUBEVIRT_DIR}/manifests/testing/rook-ceph/operator.yaml
    _kubectl apply -f ${KUBEVIRT_DIR}/manifests/testing/rook-ceph/cluster.yaml
    _kubectl apply -f ${KUBEVIRT_DIR}/manifests/testing/rook-ceph/pool.yaml

    # wait for ceph
    until _kubectl get cephblockpools -n rook-ceph replicapool -o jsonpath='{.status.phase}' | grep Ready; do
        ((count++)) && ((count == 120)) && echo "Ceph not ready in time" && exit 1
        echo "Error waiting for Ceph to be Ready, sleeping 5s and retrying"
        sleep 5
    done
fi

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

if [[ "$KUBEVIRT_PROVIDER" =~ os-* ]] || [[ "$KUBEVIRT_PROVIDER" =~ (okd|ocp)-* ]]; then
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

# Ensure the KubeVirt API is available
count=0
until _kubectl api-resources --api-group=kubevirt.io | grep kubevirts; do
    ((count++)) && ((count == 30)) && echo "KubeVirt API not found" && exit 1
    echo "waiting for KubeVirt API"
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

# Wait until KubeVirt is ready
count=0
until _kubectl wait -n kubevirt kv kubevirt --for condition=Available --timeout 5m; do
    ((count++)) && ((count == 5)) && echo "KubeVirt not ready in time" && dump_kubevirt && exit 1
    echo "Error waiting for KubeVirt to be Available, sleeping 1m and retrying"
    sleep 1m
done

echo "Done"
