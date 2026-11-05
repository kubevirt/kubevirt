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
# Copyright The KubeVirt Authors.
#

set -e
set -o pipefail

# KUBEVIRT_USE_DRA_CPU is the single switch for the driver, so that calling this script from a
# cluster that does not want it is a no-op.
if [[ "${KUBEVIRT_USE_DRA_CPU:-false}" != "true" ]]; then
    echo "KUBEVIRT_USE_DRA_CPU is not true, not installing dra-driver-cpu"
    exit 0
fi

: "${KUBEVIRT_PROVIDER:?FATAL: missing KUBEVIRT_PROVIDER}"

# The driver's grouped device mode, which KubeVirt's CPU claims rely on, needs
# DRAConsumableCapacity. That is only beta and on by default from 1.36; below it the claims are
# never allocated and VMIs just sit Pending, so fail here instead.
provider_version="${KUBEVIRT_PROVIDER##*-}"
if [[ "${provider_version}" =~ ^[0-9]+\.[0-9]+$ ]] &&
    [[ "$(printf '%s\n' "1.36" "${provider_version}" | sort -V | head -n1)" != "1.36" ]]; then
    echo "FATAL: ${KUBEVIRT_PROVIDER} predates k8s 1.36, where DRAConsumableCapacity became beta" >&2
    exit 1
fi

SCRIPT_PATH="$(dirname "$(realpath "$0")")"
# ${SCRIPT_PATH}/../../.. is the kubevirtci directory, which is where hack/config-kubevirtci.sh
# puts _ci-configs. Resolved without realpath so this still works before cluster-up has run.
: "${KUBEVIRTCI_CONFIG_PATH:="$(cd "${SCRIPT_PATH}/../../.." && pwd)/_ci-configs"}"
source "${SCRIPT_PATH}/../../../hack/ensure_helm.sh"

# The chart installs the driver DaemonSet plus the cluster-scoped "dra.cpu" DeviceClass that
# KubeVirt's synthesized CPU ResourceClaims reference.
DRA_CPU_CHART=${DRA_CPU_CHART:-"oci://registry.k8s.io/dra-driver-cpu/charts/dra-driver-cpu"}
DRA_CPU_CHART_VERSION=${DRA_CPU_CHART_VERSION:-"0.3.0"}
DRA_CPU_RELEASE_NAME=${DRA_CPU_RELEASE_NAME:-dra-driver-cpu}
DRA_CPU_DRIVER_NAMESPACE=${DRA_CPU_DRIVER_NAMESPACE:-dra-driver-cpu}

# CPUs the driver never hands out, left for kubelet, the CRI and the KubeVirt infra pods.
DRA_CPU_RESERVED_CPUS=${DRA_CPU_RESERVED_CPUS:-"0-1"}

KUBECONFIG="${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/.kubeconfig"
export KUBECONFIG
KUBECTL="${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/.kubectl --kubeconfig=${KUBECONFIG}"

function _kubectl() {
    ${KUBECTL} "$@"
}

function create_privileged_driver_namespace() {
    _kubectl get namespace "${DRA_CPU_DRIVER_NAMESPACE}" >/dev/null 2>&1 && return 0

    _kubectl create namespace "${DRA_CPU_DRIVER_NAMESPACE}"
    _kubectl label namespace "${DRA_CPU_DRIVER_NAMESPACE}" \
        pod-security.kubernetes.io/enforce=privileged \
        pod-security.kubernetes.io/warn=privileged \
        pod-security.kubernetes.io/audit=privileged
}

function install_driver() {
    local helm_bin
    helm_bin="$(ensure_helm)"

    local -a helm_args=(
        upgrade -i "${DRA_CPU_RELEASE_NAME}" "${DRA_CPU_CHART}"
        --version "${DRA_CPU_CHART_VERSION}"
        --kubeconfig "${KUBECONFIG}"
        --namespace "${DRA_CPU_DRIVER_NAMESPACE}"
        --set-string "driverConfig.reservedCPUs=${DRA_CPU_RESERVED_CPUS}"
        --wait --timeout 5m
    )

    # Left at the chart default (numanode) unless asked otherwise, so a claim's CPUs always come
    # from a single NUMA cell. Only meaningful when the nodes have more than one cell.
    if [ -n "${DRA_CPU_GROUP_BY:-}" ]; then
        helm_args+=(--set-string "driverConfig.groupBy=${DRA_CPU_GROUP_BY}")
    fi

    "${helm_bin}" "${helm_args[@]}"
}

function wait_for_resource_slices() {
    local expected_count nodes_with_slices
    expected_count=$(_kubectl get nodes --no-headers | wc -l | tr -d ' ')

    for _ in $(seq 1 60); do
        nodes_with_slices=$(_kubectl get resourceslices \
            --field-selector "spec.driver=dra.cpu" \
            -o jsonpath='{range .items[*]}{.spec.nodeName}{"\n"}{end}' 2>/dev/null \
            | sort -u | grep -c . || true)
        if [ "${nodes_with_slices}" -ge "${expected_count}" ]; then
            echo "dra.cpu published ResourceSlices on ${nodes_with_slices} node(s)"
            return 0
        fi
        echo "Waiting for dra.cpu ResourceSlices (${nodes_with_slices}/${expected_count} nodes)..."
        sleep 5
    done

    echo "FATAL: dra.cpu did not publish ResourceSlices for ${expected_count} node(s)" >&2
    _kubectl get resourceslices -o wide >&2 || true
    _kubectl logs -n "${DRA_CPU_DRIVER_NAMESPACE}" -l app.kubernetes.io/name=dra-driver-cpu --tail=100 >&2 || true
    return 1
}

function main() {
    echo "===== Installing dra-driver-cpu ====="
    create_privileged_driver_namespace
    install_driver

    echo ""
    echo "===== Waiting for dra-driver-cpu to publish its CPU topology ====="
    wait_for_resource_slices

    echo ""
    echo "===== dra-driver-cpu is ready ====="
    _kubectl get deviceclass dra.cpu
    _kubectl get pods -n "${DRA_CPU_DRIVER_NAMESPACE}"
    _kubectl get resourceslices -o wide
}

main "$@"
