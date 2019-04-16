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
# Copyright 2017 Red Hat, Inc.
#

source hack/common.sh

# create namespaces
"${CMD}" create ns kubevirt
"${CMD}" create ns cdi
"${CMD}" create ns kubevirt-hyperconverged
"${CMD}" create ns cluster-network-addons-operator

if [ "${CMD}" == "kubectl" ]; then
    # switch namespace to kubevirt
    kubectl config set-context $(kubectl config current-context) --namespace=kubevirt-hyperconverged
else
    # Switch project to kubevirt
    oc project kubevirt-hyperconverged
fi
# Deploy HCO manifests
"${CMD}" create -f deploy/standard/crds/hco.crd.yaml
"${CMD}" create -f deploy/standard/

# Create kubevirt-operator
"${CMD}" create -f "${KUBEVIRT_OPERATOR_URL}" || true

# Create cdi-operator
"${CMD}" create -f "${CDI_OPERATOR_URL}" || true

# Create cluster-network-addons-operator
"${CMD}" create -f "${CNA_URL_PREFIX}"/network-addons-config.crd.yaml
"${CMD}" create -f "${CNA_URL_PREFIX}"/operator.yaml
"${CMD}" create -f "${CNA_URL_PREFIX}"/network-addons-config-example.cr.yaml

# Create an HCO CustomResource
"${CMD}" create -f deploy/standard/crds/hco.cr.yaml

# Wait for all the operators to be ready
"${CMD}" wait deployment/hyperconverged-cluster-operator --for=condition=available
