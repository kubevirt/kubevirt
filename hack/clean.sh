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

# Remove HCO
"${CMD}" delete -f deploy/standard/crds/hco.cr.yaml --wait=false --ignore-not-found || true
"${CMD}" wait --for=delete hyperconverged.hco.kubevirt.io/hyperconverged-cluster || true
"${CMD}" delete -f deploy/standard/crds/hco.crd.yaml --wait=false --ignore-not-found || true
"${CMD}" delete -f deploy/standard/ --ignore-not-found || true
"${CMD}" delete ns kubevirt-hyperconverged --ignore-not-found || true

# Delete kubevirt-operator
"${CMD}" delete -n kubevirt apiservice v1alpha3.kubevirt.io --wait=false --ignore-not-found || true
"${CMD}" delete -f "${KUBEVIRT_OPERATOR_URL}" --ignore-not-found || true

# Delete cdi-operator
"${CMD}" delete -n cdi apiservice v1alpha1.cdi.kubevirt.io --wait=false --ignore-not-found || true
"${CMD}" delete -f "${CDI_OPERATOR_URL}" --ignore-not-found || true

# Delete cna-operator
"${CMD}" delete -f "${CNA_URL_PREFIX}"/network-addons-config.crd.yaml --ignore-not-found || true
"${CMD}" delete -f "${CNA_URL_PREFIX}"/operator.yaml --ignore-not-found || true
"${CMD}" delete ns cluster-network-addons-operator --ignore-not-found || true

# Delete ssp-operator
"${CMD}" delete -f "${SSP_URL_PREFIX}"/kubevirt-ssp-operator-crd.yaml --ignore-not-found || true
"${CMD}" delete -f "${SSP_URL_PREFIX}"/kubevirt-ssp-operator.yaml --ignore-not-found || true

# Delete kubevirt-web-ui
"${CMD}" delete -f "${KWEBUI_URL_PREFIX}"/crds/kubevirt_v1alpha1_kwebui_crd.yaml -n kubevirt-web-ui || true
"${CMD}" delete -f "${KWEBUI_URL_PREFIX}"/operator.yaml -n kubevirt-web-ui || true
"${CMD}" delete ns kubevirt-web-ui || true
