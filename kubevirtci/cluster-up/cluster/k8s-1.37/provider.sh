#!/usr/bin/env bash
set -e

if [ "${KUBEVIRT_WITH_SRIOV}" == "true" ]; then
    if [ "${KUBEVIRT_WITH_CNAO}" != "true" ]; then
        export KUBEVIRT_WITH_MULTUS=true
    fi
    export KUBEVIRT_NUM_NUMA_NODES=2
fi

# shellcheck disable=SC1090
source "${KUBEVIRTCI_PATH}/cluster/k8s-provider-common.sh"
