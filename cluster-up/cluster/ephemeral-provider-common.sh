#!/usr/bin/env bash

set -e

KUBEVIRT_WITH_ETC_IN_MEMORY=${KUBEVIRT_WITH_ETC_IN_MEMORY:-false}
KUBEVIRT_WITH_ETC_CAPACITY=${KUBEVIRT_WITH_ETC_CAPACITY:-none}
KUBEVIRTCI_VERBOSE=${KUBEVIRTCI_VERBOSE:-true}

if [ -z "${KUBEVIRTCI_TAG}" ] && [ -z "${KUBEVIRTCI_GOCLI_CONTAINER}" ]; then
    echo "FATAL: either KUBEVIRTCI_TAG or KUBEVIRTCI_GOCLI_CONTAINER must be set"
    exit 1
fi

if [ -n "${KUBEVIRTCI_TAG}" ] && [ -n "${KUBEVIRTCI_GOCLI_CONTAINER}" ]; then
    echo "WARNING: KUBEVIRTCI_GOCLI_CONTAINER is set and will take precedence over the also set KUBEVIRTCI_TAG"
fi

if [ "${KUBEVIRTCI_RUNTIME}" = "podman" ]; then
    _cri_bin=podman
    _docker_socket="${HOME}/podman.sock"
elif [ "${KUBEVIRTCI_RUNTIME}" = "docker" ]; then
    _cri_bin=docker
    _docker_socket="/var/run/docker.sock"
else
    if curl --unix-socket /${HOME}/podman.sock http://d/v3.0.0/libpod/info >/dev/null 2>&1; then
        _cri_bin=podman
        _docker_socket="${HOME}/podman.sock"
        [ "$KUBEVIRTCI_VERBOSE" = 'true' ] && echo "selecting podman as container runtime"
    elif docker ps >/dev/null; then
        _cri_bin=docker
        _docker_socket="/var/run/docker.sock"
        [ "$KUBEVIRTCI_VERBOSE" = 'true' ] && echo "selecting docker as container runtime"
    else
        echo "no working container runtime found. Neither docker nor podman seems to work."
        exit 1
    fi
fi

_cli_container="${KUBEVIRTCI_GOCLI_CONTAINER:-quay.io/kubevirtci/gocli:${KUBEVIRTCI_TAG}}"
_cli="${_cri_bin} run --privileged --net=host --rm ${USE_TTY} -v ${_docker_socket}:/var/run/docker.sock"
# gocli will try to mount /lib/modules to make it accessible to dnsmasq in
# in case it exists
if [ -d /lib/modules ]; then
    _cli="${_cli} -v /lib/modules/:/lib/modules/"
fi
_cli="${_cli} ${_cli_container}"

function _main_ip() {
    echo 127.0.0.1
}

function _port() {
    # shellcheck disable=SC2154
    ${_cli} ports --prefix $provider_prefix "$@"
}

function prepare_config() {
    BASE_PATH=${KUBEVIRTCI_CONFIG_PATH:-$PWD}
    cat >$BASE_PATH/$KUBEVIRT_PROVIDER/config-provider-$KUBEVIRT_PROVIDER.sh <<EOF
master_ip=$(_main_ip)
kubeconfig=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
kubectl=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubectl
gocli=${BASE_PATH}/../cluster-up/cli.sh
docker_prefix=\${DOCKER_PREFIX:-localhost:$(_port registry)/kubevirt}
manifest_docker_prefix=\${DOCKER_PREFIX:-registry:5000/kubevirt}
EOF
}

function _registry_volume() {
    # shellcheck disable=SC2154
    echo ${job_prefix}_registry
}

function _add_common_params() {
    # shellcheck disable=SC2155
    local params="--nodes ${KUBEVIRT_NUM_NODES} --memory ${KUBEVIRT_MEMORY_SIZE} --cpu 6 --secondary-nics ${KUBEVIRT_NUM_SECONDARY_NICS} --random-ports --background --prefix $provider_prefix --registry-volume $(_registry_volume) ${KUBEVIRT_PROVIDER} ${KUBEVIRT_PROVIDER_EXTRA_ARGS}"
    if [[ $TARGET =~ windows.* ]] && [ -n "$WINDOWS_NFS_DIR" ]; then
        params=" --nfs-data $WINDOWS_NFS_DIR $params"
    elif [[ $TARGET =~ os-.* ]] && [ -n "$RHEL_NFS_DIR" ]; then
        params=" --nfs-data $RHEL_NFS_DIR $params"
    fi
    if [ -n "${KUBEVIRTCI_PROVISION_CHECK}" ]; then
        params=" --container-registry=quay.io --container-suffix=:latest $params"
    fi

    if [ $KUBEVIRT_WITH_ETC_IN_MEMORY == "true" ]; then
        params=" --run-etcd-on-memory $params"
        if [ $KUBEVIRT_WITH_ETC_CAPACITY != "none" ]; then
            params=" --etcd-capacity $KUBEVIRT_WITH_ETC_CAPACITY $params"
        fi
    fi
    if [ $KUBEVIRT_DEPLOY_ISTIO == "true" ]; then
        params=" --enable-istio $params"
    fi

    # alternate (new) way to specify storage providers
    if [[ $KUBEVIRT_STORAGE == "rook-ceph-default" ]] && [[ $KUBEVIRT_PROVIDER_EXTRA_ARGS != *"--enable-ceph"* ]]; then
        params=" --enable-ceph $params"
    fi

    if [[ $KUBEVIRT_DEPLOY_PROMETHEUS == "true" ]] &&
        [[ $KUBEVIRT_PROVIDER_EXTRA_ARGS != *"--enable-prometheus"* ]]; then

        if [[ ($KUBEVIRT_PROVIDER =~ k8s-1\.1.*) || ($KUBEVIRT_PROVIDER =~ k8s-1.20) ]]; then
            echo "ERROR: cluster up failed because prometheus is only supported for providers >= k8s-1.21\n"
            echo "the current provider is $KUBEVIRT_PROVIDER, consider updating to a newer version, or\n"
            echo "disabling Prometheus using export KUBEVIRT_DEPLOY_PROMETHEUS=false"
            exit 1
        fi

        params=" --enable-prometheus $params"

        if [[ $KUBEVIRT_DEPLOY_PROMETHEUS_ALERTMANAGER == "true" ]] &&
            [[ $KUBEVIRT_PROVIDER_EXTRA_ARGS != *"--enable-grafana"* ]]; then
            params=" --enable-prometheus-alertmanager $params"
        fi

        if [[ $KUBEVIRT_DEPLOY_GRAFANA == "true" ]] &&
            [[ $KUBEVIRT_PROVIDER_EXTRA_ARGS != *"--enable-grafana"* ]]; then
            params=" --enable-grafana $params"
        fi
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
