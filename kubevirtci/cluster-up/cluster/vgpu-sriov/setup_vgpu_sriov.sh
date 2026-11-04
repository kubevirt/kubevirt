#!/usr/bin/env bash
# Build and load the synthetic SR-IOV NVIDIA vGPU modules on VM workers.
set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd "$script_dir/../../.." && pwd)
export KUBEVIRTCI_PATH="$repo_root/cluster-up/"
export KUBEVIRTCI_CONFIG_PATH="${KUBEVIRTCI_CONFIG_PATH:-$repo_root/../_ci-configs}"
export KUBEVIRT_PROVIDER="${KUBEVIRT_PROVIDER:-k8s-1.37}"
export KUBECONFIG="${KUBECONFIG:-$KUBEVIRTCI_CONFIG_PATH/$KUBEVIRT_PROVIDER/.kubeconfig}"

if [[ "$KUBEVIRT_PROVIDER" != k8s-1.37 ]]; then
    echo "This module setup currently supports k8s-1.37 VM workers only" >&2
    exit 1
fi

if [[ -z "${KUBEVIRTCI_TAG:-}" && -z "${KUBEVIRTCI_GOCLI_CONTAINER:-}" ]]; then
    KUBEVIRTCI_TAG=$(curl --fail --silent --show-error --location \
        https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest)
    export KUBEVIRTCI_TAG
fi

ssh_node() { "$repo_root/cluster-up/ssh.sh" "$@"; }

run_step() {
    local label=$1 log
    shift
    log=$(mktemp)
    printf '  %-30s' "$label"
    if "$@" >"$log" 2>&1; then
        echo OK
    else
        echo FAILED >&2
        cat "$log" >&2
        rm -f "$log"
        return 1
    fi
    rm -f "$log"
}

copy_sources() {
    local node=$1 ip=$2
    "$runtime" cp "$archive" "${KUBEVIRT_PROVIDER}-${node}:/tmp/vgpu-sriov.tar.gz"
    "$runtime" exec "${KUBEVIRT_PROVIDER}-${node}" scp \
        -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
        -i /vagrant.key /tmp/vgpu-sriov.tar.gz "root@${ip}:/tmp/vgpu-sriov.tar.gz"
    ssh_node "$node" 'sudo mkdir -p /tmp/vgpu-sriov && sudo tar -xzf /tmp/vgpu-sriov.tar.gz -C /tmp/vgpu-sriov'
}

setup_node() {
    local node=$1 ip=$2
    echo "$node:"
    run_step 'Copy module source' copy_sources "$node" "$ip"
    run_step 'Install build dependencies' ssh_node "$node" 'bash /tmp/vgpu-sriov/configure_node.sh deps'
    run_step 'Build kernel modules' ssh_node "$node" 'bash /tmp/vgpu-sriov/configure_node.sh build'
    run_step 'Load kernel modules' ssh_node "$node" 'bash /tmp/vgpu-sriov/configure_node.sh load'
    run_step 'Verify PF and VFs' ssh_node "$node" 'bash /tmp/vgpu-sriov/configure_node.sh verify'
    run_step 'Label worker node' kubectl label node "$node" vgpu_sriov_capable=true --overwrite
}

status_node() {
    echo "$1:"
    # ssh.sh joins the remote command into one line; keep shell separators explicit.
    # shellcheck disable=SC2016
    ssh_node "$1" 'set -e;
        lspci -Dnn | grep "faca:";
        for fn in 1 2 3 4; do
            vf="/sys/bus/pci/devices/faca:00:00.$fn";
            test -d "$vf";
            profile=$(cat "$vf/nvidia/current_vgpu_type");
            driver=$(readlink "$vf/driver");
            printf "%s profile=%s driver=%s\n" "${vf##*/}" "$profile" "${driver##*/}";
        done'
}

if (( $# > 2 )); then
    echo "Usage: $0 {setup|status} [node]" >&2
    exit 2
fi
command=${1:-setup}
case "$command" in
    setup|status) ;;
    *) echo "Usage: $0 {setup|status} [node]" >&2; exit 2 ;;
esac

if [[ -n "${2:-}" ]]; then
    nodes=$2
else
    nodes=$(kubectl get nodes -l node-role.kubernetes.io/worker -o custom-columns=NAME:.metadata.name --no-headers)
fi
[[ -n "$nodes" ]] || { echo 'No worker nodes found' >&2; exit 1; }

if [[ "$command" == setup ]]; then
    archive=$(mktemp /tmp/vgpu-sriov.XXXXXX.tar.gz)
    trap 'rm -f "$archive"' EXIT
    tar -czf "$archive" -C "$script_dir" \
        configure_node.sh \
        fake-iommu/{Makefile,fake-iommu.c} \
        fake-pci/{Makefile,fake-pci.c} \
        fake-nvidia-vgpu/{Makefile,fake-nvidia-sriov.c}
    if [[ -n "${KUBEVIRTCI_RUNTIME:-}" ]]; then
        runtime=$KUBEVIRTCI_RUNTIME
    elif podman ps >/dev/null 2>&1; then
        runtime=podman
    else
        runtime=docker
    fi
fi

for node in $nodes; do
    if [[ "$command" == status ]]; then
        status_node "$node"
    else
        ip=$(kubectl get node "$node" -o jsonpath='{.status.addresses[?(@.type=="InternalIP")].address}')
        setup_node "$node" "$ip"
    fi
done
