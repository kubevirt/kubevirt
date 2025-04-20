#!/usr/bin/env bash
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
# Copyright The KubeVirt Authors.
#

#
# This script should be executed through the makefile via `make conformance`.

set -eExuo pipefail

echo 'Preparing directory for artifacts'
export ARTIFACTS=_out/artifacts/conformance
mkdir -p ${ARTIFACTS}

echo 'Obtaining KUBECONFIG of the development cluster'
export KUBECONFIG=$(./kubevirtci/cluster-up/kubeconfig.sh)

sonobuoy_args="--wait --plugin _out/manifests/release/conformance.yaml"

add_to_label_filter() {
    local label=$1
    local separator=$2
    if [[ -z $label_filter ]]; then
        label_filter="${label}"
    else
        label_filter="${label_filter}${separator}${label}"
    fi
}

label_filter="(conformance)"

if [[ ! -z "$DOCKER_PREFIX" ]]; then
    sonobuoy_args="${sonobuoy_args} --plugin-env kubevirt-conformance.CONTAINER_PREFIX=${DOCKER_PREFIX}"
fi

if [[ ! -z "$DOCKER_TAG" ]]; then
    sonobuoy_args="${sonobuoy_args} --plugin-env kubevirt-conformance.CONTAINER_TAG=${DOCKER_TAG}"
fi

if [[ ! -z "$KUBEVIRT_E2E_FOCUS" ]]; then
    add_to_label_filter "(${KUBEVIRT_E2E_FOCUS})" "&&"
fi

if [[ ! -z "$SKIP_OUTSIDE_CONN_TESTS" ]]; then
    add_to_label_filter "(!RequiresOutsideConnectivity)" "&&"
fi

if [[ ! -z "$RUN_ON_ARM64_INFRA" ]]; then
    add_to_label_filter "(!(RequiresOutsideConnectivity && IPv6))" "&&"
fi

if [[ ! -z "$SKIP_BLOCK_STORAGE_TESTS" ]]; then
    add_to_label_filter "(!RequiresBlockStorage)" "&&"
fi

if [[ ! -z "$SKIP_SNAPSHOT_STORAGE_TESTS" ]]; then
    add_to_label_filter "(!RequiresSnapshotStorageClass)" "&&"
fi

if [[ ! -z "$KUBEVIRT_PROVIDER" ]]; then
    sonobuoy_args="${sonobuoy_args} --plugin-env kubevirt-conformance.KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER}"
fi

if [[ ! -z $label_filter ]]; then
    sonobuoy_args="${sonobuoy_args} --plugin-env kubevirt-conformance.E2E_LABEL=${label_filter}"
fi

echo 'Executing conformance tests and wait for them to finish'
echo "Using $sonobuoy_args as arguments to sonobuoy"
sonobuoy run ${sonobuoy_args}

trap "{ echo 'Cleaning up after the test execution'; sonobuoy delete --wait; }" EXIT SIGINT SIGTERM SIGQUIT

echo 'Downloading report about the test execution'
results_archive=${ARTIFACTS}/$(cd ${ARTIFACTS} && sonobuoy retrieve)

echo 'Results:'
RES=$(sonobuoy results ${results_archive})
echo "$RES"

echo "Extracting the full report and keep it under ${ARTIFACTS}"
tar xf ${results_archive} -C ${ARTIFACTS}
cp ${ARTIFACTS}/plugins/kubevirt-conformance/results/global/junit.xml ${ARTIFACTS}/junit.conformance.xml

echo 'Evaluating success of the test run'
if ! echo "$RES" | grep -q 'Status: passed'; then
    echo 'Conformance suite has failed'
    exit 1
fi

echo 'Conformance suite has successfully completed'
