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
"${CMD}" delete -f deploy/standard/crds/hco.cr.yaml --wait=false
"${CMD}" wait --for=delete hyperconverged.hco.kubevirt.io/hyperconverged-cluster
"${CMD}" delete -f deploy/standard/crds/hco.crd.yaml --wait=false
"${CMD}" delete -f deploy/standard/
"${CMD}" delete ns kubevirt-hyperconverged

# Delete kubevirt-operator
"${CMD}" delete -n kubevirt apiservice v1alpha3.kubevirt.io --wait=false
"${CMD}" delete -f "${KUBEVIRT_OPERATOR_URL}"

# # Delete cdi-operator
"${CMD}" delete -n cdi apiservice v1alpha1.cdi.kubevirt.io --wait=false
"${CMD}" delete -f "${CDI_OPERATOR_URL}"

# Delete cna-operator
"${CMD}" delete -f "${CNA_URL_PREFIX}"/network-addons-config.crd.yaml
"${CMD}" delete -f "${CNA_URL_PREFIX}"/operator.yaml
"${CMD}" delete ns cluster-network-addons-operator
