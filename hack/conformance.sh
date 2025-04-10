#!/usr/bin/env bash
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

# skip following tests on Arm64 conformance test lane
# 1. skip outside connection test: outside connection not works in current Arm64 CI infra
# 2. skip CDI related tests: we need to verify the CDI setup in arm64 e2e test environment
# 3. skip ACPI related tests: ACPI is not support on Arm
# 4. skip tests that use NewAlpineWithTestTooling image: currently we do not have Arm64 version NewAlpineWithTestTooling image
# 5. skip watchdog related tests: watchdog devices is not support by the qemu-kvm in the arm64 version virt-launcher
# 6. skip SATA related tests: SATA bus is not support by the qemu-kvm in the arm64 version virt-launcher
if [[ ! -z "$RUN_ON_ARM64_INFRA" ]]; then
    add_to_label_filter "(!RequiresOutsideConnectivity)&&(!RequiresBlockStorage)&&(!RequiresSnapshotStorageClass)&&(!storage-req)&&(!ACPI)&&(!WgArm64Invalid)" "&&"
    sonobuoy_args="${sonobuoy_args} --plugin-env kubevirt-conformance.E2E_SKIP=Alpine"
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
