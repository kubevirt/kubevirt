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
"${CMD}" delete -f _out/hco.cr.yaml --ignore-not-found || true
"${CMD}" wait --for=delete hyperconverged.hco.kubevirt.io/hyperconverged-cluster || true
# TODO: delete hangs on machine.crd.yaml. Only delete the ones that don't hang
# from _out/crds/.
"${CMD}" delete -f _out/crds/hco.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/cdi.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/cna.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/common-template-bundles.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/kubevirt.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/metrics-aggregation.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/mro.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/node-labeller-bundles.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/nodemaintenance.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/template-validator.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/crds/v2vvmware.crd.yaml --ignore-not-found || true
"${CMD}" delete -f _out/operator.yaml --ignore-not-found || true

# Delete kubevirt-operator
"${CMD}" delete -n kubevirt apiservice v1alpha3.kubevirt.io --ignore-not-found || true
"${CMD}" delete -f "${KUBEVIRT_OPERATOR_URL}" --ignore-not-found || true

# Delete cdi-operator
"${CMD}" delete -n cdi apiservice v1alpha1.cdi.kubevirt.io --ignore-not-found || true
"${CMD}" delete -f "${CDI_OPERATOR_URL}" --ignore-not-found || true

# Delete cna-operator
"${CMD}" delete -f "${CNA_URL_PREFIX}"/network-addons-config.crd.yaml --ignore-not-found || true
"${CMD}" delete -f "${CNA_URL_PREFIX}"/operator.yaml --ignore-not-found || true
"${CMD}" delete ns cluster-network-addons-operator --ignore-not-found || true

# Delete ssp-operator
"${CMD}" delete -f "${SSP_URL_PREFIX}"/kubevirt-ssp-operator-crd.yaml --ignore-not-found || true
"${CMD}" delete -f "${SSP_URL_PREFIX}"/kubevirt-ssp-operator.yaml --ignore-not-found || true

# Remove other settings
"${CMD}" delete -f _out/cluster_role_binding.yaml --ignore-not-found || true
"${CMD}" delete -f _out/cluster_role.yaml --ignore-not-found || true
"${CMD}" delete -f _out/service_account.yaml --ignore-not-found || true

# Delete namespace at the end
# "${CMD}" delete ns kubevirt-hyperconverged --ignore-not-found || true
