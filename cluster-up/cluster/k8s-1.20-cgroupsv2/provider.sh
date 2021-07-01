#!/usr/bin/env bash
set -e

export KUBEVIRT_PROVIDER=k8s-1.20
export KUBEVIRT_PROVIDER_EXTRA_ARGS="${KUBEVIRT_PROVIDER_EXTRA_ARGS} --kernel-args='systemd.unified_cgroup_hierarchy=1'"

# shellcheck disable=SC1090
source "${KUBEVIRTCI_PATH}/cluster/k8s-provider-common.sh"
