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
    export KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-docker.io/quiquell/kindest_node:v1.19.12@sha256:e4b640e4a85910b734f13c624ef714d8244a7ad2f20b6b5ca92cd33b91f1cc14}"
    export KUBECTL_PATH="${KUBECTL_PATH:-/bin/kubectl}"
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

    echo "$KUBEVIRT_PROVIDER cluster '$CLUSTER_NAME' is ready"
}

set_kind_params

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh
