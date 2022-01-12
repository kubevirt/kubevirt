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
# Copyright 2020 Red Hat, Inc.
#

set -ex

function test_delete_ns(){
    if [ "${CMD}" == "oc" ]; then
        echo "Trying to delete kubevirt-hyperconverged namespace when the hyperconverged CR is still there"
        # this should fail with a clear error message
        DELETE_ERROR_TEXT="$(${CMD} delete namespace kubevirt-hyperconverged 2>&1 || true)"

        # try to mitigate CI flakiness when we randomly get
        # "x509: certificate signed by unknown authority" errors
        if [[ $DELETE_ERROR_TEXT == *"x509: certificate signed by unknown authority"* ]]; then
          # gave it time to recovery
          sleep 300
          DELETE_ERROR_TEXT="$(${CMD} delete namespace kubevirt-hyperconverged 2>&1 || true)"
        fi
        # and eventually try again...
        if [[ $DELETE_ERROR_TEXT == *"x509: certificate signed by unknown authority"* ]]; then
          sleep 300
          DELETE_ERROR_TEXT="$(${CMD} delete namespace kubevirt-hyperconverged 2>&1 || true)"
        fi

        echo "${DELETE_ERROR_TEXT}" | grep "denied the request: HyperConverged CR is still present, please remove it before deleting the containing namespace"

        echo "kubevirt-hyperconverged namespace should be still there"
        ${CMD} get namespace kubevirt-hyperconverged -o yaml

    else
        echo "Ignoring webhook on k8s where we don't have OLM based validating webhooks"
    fi

    # TODO: workaround for https://bugzilla.redhat.com/show_bug.cgi?id=2037312
    # remove once fixed
    echo "Explictly disabling CommonBootImageImport to be able to safely remove the product"
    ./hack/retry.sh 10 3 "${CMD} patch hco -n kubevirt-hyperconverged kubevirt-hyperconverged --type=json kubevirt-hyperconverged -p '[{ \"op\": \"replace\", \"path\": \"/spec/featureGates/enableCommonBootImageImport\", \"value\": false }]'"
    sleep 10
    ./hack/retry.sh 10 30 "[[ $(${CMD} get DataImportCron -A --no-headers | wc -l) -eq 0 ]]"
    # ---

    echo "Delete the hyperconverged CR to remove the product"
    timeout 10m ${CMD} delete hyperconverged -n kubevirt-hyperconverged kubevirt-hyperconverged

    # TODO: workaround for https://bugzilla.redhat.com/show_bug.cgi?id=2031919
    # remove once fixed
    INFRASTRUCTURETOPOLOGY=$(${CMD} get infrastructure.config.openshift.io cluster -o json | jq -j '.status.infrastructureTopology')
    if [[ "${INFRASTRUCTURETOPOLOGY}" == "SingleReplica" ]]; then
      echo "Explictly deleting kubevirt apiservice leftovers"
      ${CMD} delete apiservice v1alpha3.subresources.kubevirt.io --ignore-not-found=true
      ${CMD} delete apiservice v1.subresources.kubevirt.io --ignore-not-found=true
    fi
    # ---

    echo "Finally delete kubevirt-hyperconverged namespace"
    timeout 10m ${CMD} delete namespace kubevirt-hyperconverged
}

