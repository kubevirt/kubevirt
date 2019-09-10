#!/usr/bin/env bash

set -e

image="okd-4.1@sha256:03b08bf66bf33c3ae1a1f63f1184761535513395e7b9c4cd496e22fc1eb2206b"

source ${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh

function _port() {
    ${_cli} ports --prefix $provider_prefix --container-name cluster "$@"
}

function up() {
    workers=$(($KUBEVIRT_NUM_NODES-1))
    if [[ ( $workers < 1 ) ]]; then
        workers=1
    fi
    echo "Number of workers: $workers"
    params="--random-ports --background --prefix $provider_prefix --master-cpu 6 --workers-cpu 6 --registry-volume $(_registry_volume) --workers $workers kubevirtci/${image}"
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
