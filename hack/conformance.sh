#!/usr/bin/env bash
#
# This script should be executed through the makefile via `make conformance`.

set -eExuo pipefail

echo 'Preparing directory for artifacts'
export ARTIFACTS=_out/artifacts/conformance
mkdir -p ${ARTIFACTS}

echo 'Obtaining KUBECONFIG of the development cluster'
export KUBECONFIG=$(./cluster-up/kubeconfig.sh)

sonobuoy_args="--wait --plugin _out/manifests/release/conformance.yaml"

if [[ ! -z "$SKIP_OUTSIDE_CONN_TESTS" ]]; then
    sonobuoy_args="${sonobuoy_args} --plugin-env kubevirt-conformance.E2E_SKIP=\[outside_connectivity\]"
fi

echo 'Executing conformance tests and wait for them to finish'
sonobuoy run ${sonobuoy_args}

trap "{ echo 'Cleaning up after the test execution'; sonobuoy delete --wait; }" EXIT SIGINT SIGTERM SIGQUIT

echo 'Downloading report about the test execution'
results_archive=${ARTIFACTS}/$(cd ${ARTIFACTS} && sonobuoy retrieve)

echo 'Results:'
sonobuoy results ${results_archive}

echo "Extracting the full report and keep it under ${ARTIFACTS}"
tar xf ${results_archive} -C ${ARTIFACTS}
cp ${ARTIFACTS}/plugins/kubevirt-conformance/results/global/junit.xml ${ARTIFACTS}/junit.conformance.xml

echo 'Evaluating success of the test run'
sonobuoy results ${results_archive} | grep -q 'Status: passed' || {
    echo 'Conformance suite has failed'
    exit 1
}
echo 'Conformance suite has successfully completed'
