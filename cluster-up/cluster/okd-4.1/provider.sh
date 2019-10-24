#!/usr/bin/env bash

set -e

image="okd-4.1@sha256:2b7b5e09b9bdf2ca40b8e153a111702584e2a3e802643e3e7df1f2d97eca0ce8"

source ${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh

function _port() {
    ${_cli} ports --prefix $provider_prefix --container-name cluster "$@"
}

function _install_from_cluster() {
    local src_cid="$1"
    local src_file="$2"
    local dst_perms="$3"
    local dst_file="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$4"

    touch $dst_file
    chmod $dst_perms $dst_file
    docker exec $src_cid cat $src_file > $dst_file
}


function up() {
    workers=$(($KUBEVIRT_NUM_NODES-1))
    if [[ ( $workers < 1 ) ]]; then
        workers=1
    fi
    echo "Number of workers: $workers"
    params="--random-ports --background --prefix $provider_prefix --master-cpu 6 --workers-cpu 6 --workers-memory 8192 --secondary-nics ${KUBEVIRT_NUM_SECONDARY_NICS} --registry-volume $(_registry_volume) --workers $workers kubevirtci/${image}"
    if [[ ! -z "${RHEL_NFS_DIR}" ]]; then
        params=" --nfs-data $RHEL_NFS_DIR ${params}"
    fi

    if [[ ! -z "${OKD_CONSOLE_PORT}" ]]; then
        params=" --ocp-console-port $OKD_CONSOLE_PORT ${params}"
    fi

    ${_cli} run okd ${params}

    # Copy k8s config and kubectl
    cluster_container_id=$(docker ps -f "name=$provider_prefix-cluster" --format "{{.ID}}")

    _install_from_cluster $cluster_container_id /bin/oc 0755 .kubectl
    _install_from_cluster $cluster_container_id /root/install/auth/kubeconfig 0644 .kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster test-1 --server=https://$(_main_ip):$(_port k8s)
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster test-1 --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}
