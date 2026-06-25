#!/usr/bin/env bash

set -e
set -o pipefail

SCRIPT_PATH="$(dirname "$(realpath "$0")")"
VFIO_NODE_SETUP_SCRIPT="${SCRIPT_PATH}/vfio-node/setup_node_vfio.sh"
VFIO_NODE_WORKDIR="/tmp/fake-vfio"

: "${KUBEVIRTCI_CONFIG_PATH:?FATAL: missing KUBEVIRTCI_CONFIG_PATH}"
: "${KUBEVIRT_PROVIDER:?FATAL: missing KUBEVIRT_PROVIDER}"
: "${FAKE_PCI_DEVICES:=8}"
: "${FAKE_IOMMU:=true}"

KUBECONFIG="${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/.kubeconfig"
KUBECTL="${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/.kubectl --kubeconfig=${KUBECONFIG}"

VFIO_NODE_LABEL_KEY="fake-vfio-capable"
VFIO_NODE_LABEL_VALUE="true"
VFIO_NODE_LABEL="${VFIO_NODE_LABEL_KEY}=${VFIO_NODE_LABEL_VALUE}"

source "${SCRIPT_PATH}/install_dra_example_driver.sh"

_kubectl() {
    ${KUBECTL} "$@"
}

ssh_to_node() {
    local node_name=$1
    shift
    "${SCRIPT_PATH}/../../ssh.sh" "$node_name" "$@"
}

encode_file() {
    local file=$1
    base64 <"${file}" | tr -d '\n'
}

encode_vfio_assets() {
    tar -C "${SCRIPT_PATH}" -cz fake-iommu fake-pci setup-fake-pci-host.sh | base64 | tr -d '\n'
}

validate_inputs() {
    if [ ! -d "${SCRIPT_PATH}/fake-iommu" ] || [ ! -d "${SCRIPT_PATH}/fake-pci" ]; then
        echo "FATAL: fake VFIO module sources not found under ${SCRIPT_PATH}" >&2
        exit 1
    fi
    if [ ! -x "${SCRIPT_PATH}/setup-fake-pci-host.sh" ]; then
        echo "FATAL: setup-fake-pci-host.sh not found or not executable under ${SCRIPT_PATH}" >&2
        exit 1
    fi
    if [ ! -f "${VFIO_NODE_SETUP_SCRIPT}" ]; then
        echo "FATAL: node setup script not found at ${VFIO_NODE_SETUP_SCRIPT}" >&2
        exit 1
    fi
}

copy_vfio_payload_to_node() {
    local node_name=$1
    local encoded_assets encoded_setup_script

    encoded_assets=$(encode_vfio_assets)
    encoded_setup_script=$(encode_file "${VFIO_NODE_SETUP_SCRIPT}")

    echo "Copying fake VFIO payload to ${node_name}..."
    ssh_to_node "${node_name}" "rm -rf '${VFIO_NODE_WORKDIR}' && mkdir -p '${VFIO_NODE_WORKDIR}'"
    ssh_to_node "${node_name}" "printf '%s' '${encoded_assets}' | base64 -d | tar -xz -C '${VFIO_NODE_WORKDIR}'"
    ssh_to_node "${node_name}" "printf '%s' '${encoded_setup_script}' | base64 -d > '${VFIO_NODE_WORKDIR}/setup_node_vfio.sh' && chmod +x '${VFIO_NODE_WORKDIR}/setup_node_vfio.sh'"
}

run_node_vfio_setup() {
    local node_name=$1
    ssh_to_node "${node_name}" "sudo \
        FAKE_PCI_DEVICES='${FAKE_PCI_DEVICES}' \
        FAKE_PCI_DOMAIN='${FAKE_PCI_DOMAIN:-}' \
        FAKE_PCI_VENDOR_ID='${FAKE_PCI_VENDOR_ID:-}' \
        FAKE_PCI_DEVICE_ID='${FAKE_PCI_DEVICE_ID:-}' \
        FAKE_IOMMU='${FAKE_IOMMU}' \
        bash '${VFIO_NODE_WORKDIR}/setup_node_vfio.sh'"
}

configure_node_vfio() {
    local node_name=$1

    echo "===== Configuring fake VFIO on ${node_name} ====="
    copy_vfio_payload_to_node "${node_name}"
    run_node_vfio_setup "${node_name}"

    _kubectl label node "${node_name}" "${VFIO_NODE_LABEL}" --overwrite
    echo "===== fake VFIO configuration completed on ${node_name} ====="
}

print_node_status() {
    local node_name=$1
    local status_cmd='echo "  modules:"; lsmod | grep -E "^(fake_iommu|fake_pci|vfio_pci)[[:space:]]" | sed "s/^/    /" || true; echo "  vfio-pci devices:"; found=0; for d in /sys/bus/pci/drivers/vfio-pci/*:*; do [ -e "$d" ] || continue; found=1; echo "    $(basename "$d")"; done; [ "$found" -eq 1 ] || echo "    (none)"; echo "  /dev/vfio entries:"; ls -1 /dev/vfio 2>/dev/null | sed "s/^/    /" || echo "    (none)"'

    echo ""
    echo "Node fake VFIO status for ${node_name}:"
    ssh_to_node "${node_name}" "sudo bash -c $(printf "%q" "${status_cmd}")"
}

main() {
    validate_inputs

    echo "===== Starting fake VFIO cluster configuration ====="
    worker_nodes=$(_kubectl get nodes -l node-role.kubernetes.io/worker -o custom-columns=:.metadata.name --no-headers)
    worker_nodes_array=($worker_nodes)
    worker_nodes_count=${#worker_nodes_array[@]}

    if [ "${worker_nodes_count}" -eq 0 ]; then
        echo "FATAL: no worker nodes found" >&2
        exit 1
    fi

    echo "Found ${worker_nodes_count} worker node(s): ${worker_nodes_array[*]}"

    local node
    for node in "${worker_nodes_array[@]}"; do
        configure_node_vfio "${node}"
    done

    echo ""
    echo "===== Installing vfio-gpu DRA example driver ====="
    cluster::install_dra_example_driver

    echo ""
    echo "===== Waiting for DRA driver pods to be ready ====="
    _kubectl wait pods -n dra-example-driver --all --for=condition=Ready --timeout=300s

    for node in "${worker_nodes_array[@]}"; do
        print_node_status "${node}"
    done

    echo ""
    echo "===== fake VFIO cluster configuration complete ====="
    _kubectl get nodes -l "${VFIO_NODE_LABEL}"
    _kubectl get pods -n dra-example-driver
}

main "$@"
