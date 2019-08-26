#!/usr/bin/env bash

set -e

image="okd-4.1@sha256:d452e8f910bd08b4aabe2a9b8fd82dc5984a3e95f7096b3ebd6c8ba836a5361d"

source ${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh

function _port() {
    ${_cli} ports --prefix $provider_prefix --container-name cluster "$@"
}

function up() {
    params="--random-ports --background --prefix $provider_prefix --master-cpu 6 --workers-cpu 6 --registry-volume $(_registry_volume) --workers 2 kubevirtci/${image}"
    if [[ ! -z "${RHEL_NFS_DIR}" ]]; then
        params=" --nfs-data $RHEL_NFS_DIR ${params}"
    fi

    if [[ ! -z "${OKD_CONSOLE_PORT}" ]]; then
        params=" --ocp-console-port $OKD_CONSOLE_PORT ${params}"
    fi

    ${_cli} run okd ${params}

    # Copy k8s config and kubectl
    cluster_container_id=$(docker ps -f "name=$provider_prefix-cluster" --format "{{.ID}}")
    docker cp $cluster_container_id:/bin/oc ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    docker cp $cluster_container_id:/root/install/auth/kubeconfig ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster test-1 --server=https://$(_main_ip):$(_port k8s)
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster test-1 --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}
