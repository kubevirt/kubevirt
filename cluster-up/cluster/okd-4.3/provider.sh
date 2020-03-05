#!/usr/bin/env bash

set -e

source ${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh
source ${KUBEVIRTCI_PATH}/cluster/openshift-provider-common.sh

function _port() {
    ${_cli} ports --prefix $provider_prefix --container-name cluster "$@"
}

function _install_from_cluster() {
    local src_cid="$1"
    local src_file="$2"
    local dst_perms="$3"
    local dst_name="$4"
    local dst_file="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$4"
    local pack8s_path="$5"

    if [ "$KUBEVIRTCI_RUNTIME" = "podman" ]; then
        pack8s exec $src_cid cp $src_file /tmp/$dst_name
        cp $pack8s_path/$dst_name $dst_file
    else
        touch $dst_file
        docker exec $src_cid cat $src_file > $dst_file
    fi
    chmod $dst_perms $dst_file
}

function up() {
    workers=$(($KUBEVIRT_NUM_NODES-1))
    if [[ ( $workers < 1 ) ]]; then
        workers=1
    fi
    echo "Number of workers: $workers"
    params="--random-ports --background --prefix $provider_prefix --master-cpu 6 --workers-cpu 6 --secondary-nics ${KUBEVIRT_NUM_SECONDARY_NICS} --registry-volume $(_registry_volume) --workers $workers kubevirtci/${image}"
    if [[ ! -z "${RHEL_NFS_DIR}" ]]; then
        params=" --nfs-data $RHEL_NFS_DIR ${params}"
    fi

    if [[ ! -z "${OKD_CONSOLE_PORT}" ]]; then
        params=" --ocp-console-port $OKD_CONSOLE_PORT ${params}"
    fi

    if [[ ! -z "${INSTALLER_PULL_SECRET}" ]]; then
        params=" --installer-pull-secret-file ${INSTALLER_PULL_SECRET} ${params}"
    fi

    if [ "$KUBEVIRTCI_RUNTIME" = "podman" ]; then
        pack8s_path=$(mktemp -d /tmp/pack8sXXX)
        params=" --volume $pack8s_path:/tmp ${params}"
    fi

    ${_cli} run okd ${params}

    # Copy k8s config and kubectl
    if [ "$KUBEVIRTCI_RUNTIME" = "podman" ]; then
        cluster_container_id=$(pack8s -p "$provider_prefix-cluster" show -i)
    else
        cluster_container_id=$(docker ps -f "name=$provider_prefix-cluster" --format "{{.ID}}")
    fi

    _install_from_cluster $cluster_container_id /usr/local/bin/oc 0755 .kubectl $pack8s_path
    _install_from_cluster $cluster_container_id /root/install/auth/kubeconfig 0644 .kubeconfig $pack8s_path

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster test-1 --server=https://$(_main_ip):$(_port k8s)
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster test-1 --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config

    ln_kubeconfig
}
