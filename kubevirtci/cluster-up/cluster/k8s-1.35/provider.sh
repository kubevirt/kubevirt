#!/usr/bin/env bash
set -e

kernel_args="nousershstk"
if [ "${KUBEVIRT_CGROUPV2}" == "false" ]; then
    kernel_args="systemd.unified_cgroup_hierarchy=0 ${kernel_args}"
fi
export KUBEVIRT_PROVIDER_EXTRA_ARGS="${KUBEVIRT_PROVIDER_EXTRA_ARGS} --kernel-args='${kernel_args}'"

# shellcheck disable=SC1090
source "${KUBEVIRTCI_PATH}/cluster/k8s-provider-common.sh"
