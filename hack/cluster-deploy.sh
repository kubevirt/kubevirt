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

set -ex pipefail

DOCKER_TAG=${DOCKER_TAG:-devel}
KUBEVIRT_DEPLOY_CDI=${KUBEVIRT_DEPLOY_CDI:-true}
CDI_DV_GC_DEFAULT=-1
CDI_DV_GC=${CDI_DV_GC:--1}

source hack/common.sh
# shellcheck disable=SC1090
source cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

function dump_kubevirt() {
    if [ "$?" -ne "0" ]; then
        echo "Dump kubevirt state:"
        hack/dump.sh
    fi
}

function _deploy_infra_for_tests() {
    if [[ "$KUBEVIRT_DEPLOY_CDI" == "false" ]]; then
        rm -f ${MANIFESTS_OUT_DIR}/testing/uploadproxy-nodeport.yaml \
            ${MANIFESTS_OUT_DIR}/testing/disks-images-provider.yaml
    fi

    # Deploy infra for testing first
    _kubectl create -f ${MANIFESTS_OUT_DIR}/testing
}

function _ensure_cdi_deployment() {
    # enable featuregate
    _kubectl patch cdi ${cdi_namespace:?} --type merge -p '{"spec": {"config": {"featureGates": [ "HonorWaitForFirstConsumer" ]}}}'

    # add insecure registries
    _kubectl patch cdi ${cdi_namespace} --type merge -p '{"spec": {"config": {"insecureRegistries": [ "registry:5000", "fakeregistry:5000" ]}}}'

    # Configure uploadproxy override for virtctl imageupload
    host_port=$(${KUBEVIRT_PATH}cluster-up/cli.sh ports uploadproxy | xargs)
    override="https://127.0.0.1:$host_port"
    _kubectl patch cdi ${cdi_namespace} --type merge -p '{"spec": {"config": {"uploadProxyURLOverride": "'"$override"'"}}}'

    # Configure DataVolume garbage collection
    if [[ $CDI_DV_GC != $CDI_DV_GC_DEFAULT ]]; then
        _kubectl patch cdi ${cdi_namespace} --type merge -p '{"spec": {"config": {"dataVolumeTTLSeconds": '"$CDI_DV_GC"'}}}'
    fi
}

function configure_prometheus() {
    if [[ $KUBEVIRT_DEPLOY_PROMETHEUS == "true" ]] && _kubectl get crd prometheuses.monitoring.coreos.com; then
        _kubectl patch prometheus k8s -n monitoring --type=json -p '[{"op": "replace", "path": "/spec/ruleSelector", "value":{}}, {"op": "replace", "path": "/spec/ruleNamespaceSelector", "value":{"matchLabels": {"kubevirt.io": ""}}}]'
    fi
}

trap dump_kubevirt EXIT

echo "Deploying ..."

# Create the installation namespace if it does not exist already
_kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Namespace
metadata:
  name: ${namespace:?}
  labels:
    pod-security.kubernetes.io/enforce: "privileged"
EOF

if [[ "$KUBEVIRT_PROVIDER" =~ kind.* || "$KUBEVIRT_PROVIDER" = "external" ]]; then
    # Don't install CDI and loopback devices it's crashing with dind because loopback devices are shared with the host
    export KUBEVIRT_DEPLOY_CDI=false
fi

_deploy_infra_for_tests

# TODO: Remove the 2nd condition when CDI is supported on ARM
if [[ "$KUBEVIRT_DEPLOY_CDI" != "false" ]] && [[ ${ARCHITECTURE} != *aarch64 ]]; then
    _ensure_cdi_deployment
fi

# Deploy kubevirt operator
_kubectl apply -f ${MANIFESTS_OUT_DIR}/release/kubevirt-operator.yaml

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
    ((count++)) && ((count == 5)) && echo "KubeVirt not ready in time" && exit 1
    echo "Error waiting for KubeVirt to be Available, sleeping 1m and retrying"
    sleep 1m
done

configure_prometheus

echo "Done $0"
