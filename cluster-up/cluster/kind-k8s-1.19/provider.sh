#!/usr/bin/env bash

set -e

DOCKER="${CONTAINER_RUNTIME:-docker}"

DEFAULT_CLUSTER_NAME="kind-1.19"
DEFAULT_HOST_PORT=5000
ALTERNATE_HOST_PORT=5001
export CLUSTER_NAME=${CLUSTER_NAME:-$DEFAULT_CLUSTER_NAME}

if [ "$CLUSTER_NAME" = "$DEFAULT_CLUSTER_NAME" ]; then
    export HOST_PORT=$DEFAULT_HOST_PORT
else
    export HOST_PORT=$ALTERNATE_HOST_PORT
fi

TESTS_NS="${TESTS_NS:-kubevirt-test-default1}"

function set_kind_params() {
    export KIND_VERSION="${KIND_VERSION:-0.11.1}"
    export KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-kindest/node:v1.19.11@sha256:07db187ae84b4b7de440a73886f008cf903fcf5764ba8106a9fd5243d6f32729}"
    export KUBECTL_PATH="${KUBECTL_PATH:-/bin/kubectl}"
}

function up() {
    cp $KIND_MANIFESTS_DIR/kind.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
    kind_up

    # remove the rancher.io kind default storageClass
    _kubectl delete sc standard

    nodes=$(_kubectl get nodes -o=custom-columns=:.metadata.name | awk NF)
    for node in $nodes; do
        # Create local-volume directories, which, on other providers, are pre-provisioned.
        # For more info, check https://github.com/kubevirt/kubevirtci/blob/master/cluster-provision/STORAGE.md
        for i in {1..10}; do
            mount_disk $node $i
        done
        $DOCKER exec $node bash -c "chmod -R 777 /var/local/kubevirt-storage/local-volume"
    done

    # create the `local` storage class - which functional tests assume to exist
    _kubectl apply -f $KIND_MANIFESTS_DIR/local-volume.yaml

    # Since Kind provider uses containers as nodes, the UUID on all of them will be the same,
    # and Migration by libvirt would be blocked, because migrate between the same UUID is forbidden.
    # Enable PodPreset so we can use it in order to mount a fake UUID for each launcher pod.
    podpreset::expose_unique_product_uuid_per_node "$CLUSTER_NAME" "$TESTS_NS"
}

function mount_disk() {
    local node=$1
    local idx=$2
    $DOCKER exec $node bash -c "mkdir -p /var/local/kubevirt-storage/local-volume/disk${idx}"
    $DOCKER exec $node bash -c "mkdir -p /mnt/local-storage/local/disk${idx}"
    $DOCKER exec $node bash -c "mount -o bind /var/local/kubevirt-storage/local-volume/disk${idx} /mnt/local-storage/local/disk${idx}"
}

set_kind_params

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh
source ${KUBEVIRTCI_PATH}/cluster/kind/podpreset.sh
