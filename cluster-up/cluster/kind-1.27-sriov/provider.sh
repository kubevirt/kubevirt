#!/usr/bin/env bash

set -e

DEFAULT_CLUSTER_NAME="sriov"
DEFAULT_HOST_PORT=5000
ALTERNATE_HOST_PORT=5001
export CLUSTER_NAME=${CLUSTER_NAME:-$DEFAULT_CLUSTER_NAME}

if [ $CLUSTER_NAME == $DEFAULT_CLUSTER_NAME ]; then
    export HOST_PORT=$DEFAULT_HOST_PORT
else
    export HOST_PORT=$ALTERNATE_HOST_PORT
fi

function print_available_nics() {
    # print hardware info for easier debugging based on logs
    echo 'Available NICs'
    ${CRI_BIN} run --rm --cap-add=SYS_RAWIO quay.io/phoracek/lspci@sha256:0f3cacf7098202ef284308c64e3fc0ba441871a846022bb87d65ff130c79adb1 sh -c "lspci | egrep -i 'network|ethernet'"
    echo ""
}

function set_kind_params() {
    export KIND_VERSION="${KIND_VERSION:-0.18.0}"
    export KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-kindest/node:v1.27.1@sha256:9915f5629ef4d29f35b478e819249e89cfaffcbfeebda4324e5c01d53d937b09}"
    export KUBECTL_PATH="${KUBECTL_PATH:-/bin/kubectl}"
}

function print_sriov_data() {
    nodes=$(_kubectl get nodes -o=custom-columns=:.metadata.name | awk NF)
    for node in $nodes; do
        if [[ ! "$node" =~ .*"control-plane".* ]]; then
            echo "Node: $node"
            echo "VFs:"
            ${CRI_BIN} exec $node bash -c "ls -l /sys/class/net/*/device/virtfn*"
            echo "PFs PCI Addresses:"
            ${CRI_BIN} exec $node bash -c "grep PCI_SLOT_NAME /sys/class/net/*/device/uevent"
        fi
    done
}

function configure_registry_proxy() {
    [ "$CI" != "true" ] && return

    echo "Configuring cluster nodes to work with CI mirror-proxy..."

    local -r ci_proxy_hostname="docker-mirror-proxy.kubevirt-prow.svc"
    local -r kind_binary_path="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kind"
    local -r configure_registry_proxy_script="${KUBEVIRTCI_PATH}/cluster/kind/configure-registry-proxy.sh"

    KIND_BIN="$kind_binary_path" PROXY_HOSTNAME="$ci_proxy_hostname" $configure_registry_proxy_script
}

function deploy_sriov() {
    print_available_nics
    ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/config_sriov_cluster.sh
    print_sriov_data
}

function up() {
    cp $KIND_MANIFESTS_DIR/kind.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
    export CONFIG_WORKER_CPU_MANAGER=true
    kind_up

    configure_registry_proxy

    # remove the rancher.io kind default storageClass
    _kubectl delete sc standard

    deploy_sriov
    echo "$KUBEVIRT_PROVIDER cluster '$CLUSTER_NAME' is ready"
}

set_kind_params

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh
