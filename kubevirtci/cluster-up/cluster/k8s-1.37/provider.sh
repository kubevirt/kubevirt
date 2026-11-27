#!/usr/bin/env bash
set -e

export KUBEVIRT_PROVIDER_EXTRA_ARGS="${KUBEVIRT_PROVIDER_EXTRA_ARGS:-} --kernel-args='nousershstk'"

# shellcheck disable=SC1090
source "${KUBEVIRTCI_PATH}/cluster/k8s-provider-common.sh"
