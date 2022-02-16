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

#'kubevirt-test-default1' is the default namespace of
# Kubevirt SRIOV tests where the SRIOV VM's will be created.
SRIOV_TESTS_NS="${SRIOV_TESTS_NS:-kubevirt-test-default1}"

function set_kind_params() {
    export KIND_VERSION="${KIND_VERSION:-0.11.1}"
    export KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-quay.io/kubevirtci/kindest_node:v1.19.11@sha256:cbecc517bfad65e368cd7975d1e8a4f558d91160c051d0b1d10ff81488f5fb06}"
    export KUBECTL_PATH="${KUBECTL_PATH:-/bin/kubectl}"
}

function print_sriov_data() {
    nodes="$(_kubectl get nodes -o=custom-columns=:.metadata.name | awk NF)"
    for node in $nodes; do
        if [[ ! "$node" =~ .*"control-plane".* ]]; then
            echo "Node: $node"
            echo "VFs:"
            docker exec $node bash -c "ls -l /sys/class/net/*/device/virtfn*"
            echo "PFs PCI Addresses:"
            docker exec $node bash -c "grep PCI_SLOT_NAME /sys/class/net/*/device/uevent"
        fi
    done
}

function up() {
    # print hardware info for easier debugging based on logs
    echo 'Available NICs'
    docker run --rm --cap-add=SYS_RAWIO quay.io/phoracek/lspci@sha256:0f3cacf7098202ef284308c64e3fc0ba441871a846022bb87d65ff130c79adb1 sh -c "lspci | egrep -i 'network|ethernet'"
    echo ""

    cp $KIND_MANIFESTS_DIR/kind.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
    kind_up

    # remove the rancher.io kind default storageClass
    _kubectl delete sc standard

    ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/config_sriov_cluster.sh

    # In order to support live migration on containerized cluster we need to workaround
    # Libvirt uuid check for source and target nodes.
    # To do that we create PodPreset that mounts fake random product_uuid to virt-launcher pods,
    # and kubevirt SRIOV tests namespace for the PodPreset beforehand.
    podpreset::expose_unique_product_uuid_per_node "$CLUSTER_NAME" "$SRIOV_TESTS_NS"

    print_sriov_data
    echo "$KUBEVIRT_PROVIDER cluster '$CLUSTER_NAME' is ready"
}

set_kind_params

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh
source ${KUBEVIRTCI_PATH}/cluster/kind/podpreset.sh
