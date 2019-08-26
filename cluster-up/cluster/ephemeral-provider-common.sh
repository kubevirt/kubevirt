#!/usr/bin/env bash

set -e

_cli_container="kubevirtci/gocli@sha256:a7880757e2d2755c6a784c1b64c64b096769ed3ccfac9d8e535df481731c2144"
_cli_with_tty="docker run --privileged --net=host --rm -t -v /var/run/docker.sock:/var/run/docker.sock ${_cli_container}"
_cli="docker run --privileged --net=host --rm ${USE_TTY} -v /var/run/docker.sock:/var/run/docker.sock ${_cli_container}"

function _main_ip() {
    echo 127.0.0.1
}

function _port() {
    ${_cli} ports --prefix $provider_prefix "$@"
}

function prepare_config() {
    BASE_PATH=${KUBEVIRTCI_CONFIG_PATH:-$PWD}
    cat >$BASE_PATH/$KUBEVIRT_PROVIDER/config-provider-$KUBEVIRT_PROVIDER.sh <<EOF
master_ip=$(_main_ip)
kubeconfig=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
kubectl=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubectl
gocli=${BASE_PATH}/../cluster-up/cli.sh
docker_prefix=localhost:$(_port registry)/kubevirt
manifest_docker_prefix=registry:5000/kubevirt
EOF
}

function _registry_volume() {
    echo ${job_prefix}_registry
}

function _add_common_params() {
    local params="--nodes ${KUBEVIRT_NUM_NODES} --memory ${KUBEVIRT_MEMORY_SIZE} --cpu 5 --secondary-nics ${KUBEVIRT_NUM_SECONDARY_NICS} --random-ports --background --prefix $provider_prefix --registry-volume $(_registry_volume) kubevirtci/${image} ${KUBEVIRT_PROVIDER_EXTRA_ARGS}"
    if [[ $TARGET =~ windows.* ]] && [ -n "$WINDOWS_NFS_DIR" ]; then
        params=" --nfs-data $WINDOWS_NFS_DIR $params"
    elif [[ $TARGET =~ os-.* ]] && [ -n "$RHEL_NFS_DIR" ]; then
        params=" --nfs-data $RHEL_NFS_DIR $params"
    fi
    echo $params
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl "$@"
}

function down() {
    ${_cli} rm --prefix $provider_prefix
}
