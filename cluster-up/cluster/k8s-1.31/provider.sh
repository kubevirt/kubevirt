#!/usr/bin/env bash
set -e

if [ "${KUBEVIRT_CGROUPV2}" == "true" ]; then
    export KUBEVIRT_PROVIDER_EXTRA_ARGS="${KUBEVIRT_PROVIDER_EXTRA_ARGS} --kernel-args='systemd.unified_cgroup_hierarchy=1'"
fi

# shellcheck disable=SC1090
source "${KUBEVIRTCI_PATH}/cluster/k8s-provider-common.sh"
