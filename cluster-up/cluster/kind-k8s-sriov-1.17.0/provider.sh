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

function set_kind_params() {
    export KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-kindest/node:v1.17.0}"
    export KIND_VERSION="${KIND_VERSION:-0.7.0}"
    export KUBECTL_PATH="${KUBECTL_PATH:-/kind/bin/kubectl}"
}

function up() {
    if [[ "$KUBEVIRT_NUM_NODES" -ne 2 ]]; then
        echo 'SR-IOV cluster can be only started with 2 nodes'
        exit 1
    fi

    # print hardware info for easier debugging based on logs
    echo 'Available NICs'
    docker run --rm --cap-add=SYS_RAWIO quay.io/phoracek/lspci@sha256:0f3cacf7098202ef284308c64e3fc0ba441871a846022bb87d65ff130c79adb1 sh -c "lspci | egrep -i 'network|ethernet'"
    echo ""

    cp $KIND_MANIFESTS_DIR/kind.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml

    kind_up

    # remove the rancher.io kind default storageClass
    _kubectl delete sc standard

    ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/config_sriov.sh
}

set_kind_params

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh
