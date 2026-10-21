#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBEVIRTCI_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
KUBEVIRTCI_PATH="${KUBEVIRTCI_ROOT}/cluster-up/"

export KUBEVIRTCI_PATH
export KUBEVIRTCI_CONFIG_PATH="${KUBEVIRTCI_CONFIG_PATH:-${KUBEVIRTCI_ROOT}/../_ci-configs}"
export KUBEVIRT_PROVIDER="k8s-1.37"
export KUBEVIRT_NUM_NODES=3
export KUBEVIRT_SLIM="${KUBEVIRT_SLIM:-true}"
export KUBECONFIG="${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/.kubeconfig"

if [[ -z "${KUBEVIRTCI_TAG:-}" && -z "${KUBEVIRTCI_GOCLI_CONTAINER:-}" ]]; then
    KUBEVIRTCI_TAG=$(curl --fail --silent --show-error --location \
        https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest)
    export KUBEVIRTCI_TAG
fi

KUBECTL="${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/.kubectl"

if [[ ! -x "${KUBECTL}" ]]; then
    echo "Starting a 3-node ${KUBEVIRT_PROVIDER} cluster..."
    make -C "${KUBEVIRTCI_ROOT}" cluster-up
else
    echo "Using the existing ${KUBEVIRT_PROVIDER} cluster config at ${KUBECONFIG}"
fi

if [[ ! -x "${KUBECTL}" || ! -f "${KUBECONFIG}" ]]; then
    echo "Cluster startup did not produce kubectl and kubeconfig under ${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}" >&2
    exit 1
fi

"${KUBECTL}" --kubeconfig="${KUBECONFIG}" wait --for=condition=Ready nodes --all --timeout=10m

node_count=$("${KUBECTL}" --kubeconfig="${KUBECONFIG}" get nodes --no-headers | wc -l)
if [[ "${node_count}" -ne 3 ]]; then
    echo "Expected 3 nodes in ${KUBEVIRT_PROVIDER}, found ${node_count}; refusing to continue." >&2
    exit 1
fi

versions=$("${KUBECTL}" --kubeconfig="${KUBECONFIG}" get nodes -o custom-columns=VERSION:.status.nodeInfo.kubeletVersion --no-headers)
if grep -vq '^v1\.37\.' <<<"${versions}"; then
    echo "Expected all nodes to run Kubernetes 1.37; found:" >&2
    printf '%s\n' "${versions}" >&2
    exit 1
fi

echo "Installing fake vGPU kernel module support on worker nodes..."
"${SCRIPT_DIR}/setup_vgpu_mdev.sh" full-setup

echo "3-node Kubernetes 1.37 fake vGPU node setup completed."
echo "After installing KubeVirt, run ${SCRIPT_DIR}/configure_kubevirt_mdev.sh to configure mediated devices."
"${KUBECTL}" --kubeconfig="${KUBECONFIG}" get nodes -o wide
