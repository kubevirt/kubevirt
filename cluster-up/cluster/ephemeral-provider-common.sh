#!/usr/bin/env bash

set -e

KUBEVIRT_WITH_ETC_IN_MEMORY=${KUBEVIRT_WITH_ETC_IN_MEMORY:-false}
KUBEVIRT_WITH_ETC_CAPACITY=${KUBEVIRT_WITH_ETC_CAPACITY:-none}
KUBEVIRT_DNS_HOST_PORT=${KUBEVIRT_DNS_HOST_PORT:-31111}

export KUBEVIRTCI_PODMAN_SOCKET=${KUBEVIRTCI_PODMAN_SOCKET:-"/run/podman/podman.sock"}

if [ -z "${KUBEVIRTCI_TAG}" ] && [ -z "${KUBEVIRTCI_GOCLI_CONTAINER}" ]; then
    >&2 echo "FATAL: either KUBEVIRTCI_TAG or KUBEVIRTCI_GOCLI_CONTAINER must be set"
    exit 1
fi

if [ -n "${KUBEVIRTCI_TAG}" ] && [ -n "${KUBEVIRTCI_GOCLI_CONTAINER}" ]; then
    >&2 echo "WARNING: KUBEVIRTCI_GOCLI_CONTAINER is set and will take precedence over the also set KUBEVIRTCI_TAG"
fi

detect_podman_socket() {
    if curl --unix-socket "${KUBEVIRTCI_PODMAN_SOCKET}" http://d/v3.0.0/libpod/info >/dev/null 2>&1; then
        echo "${KUBEVIRTCI_PODMAN_SOCKET}"
    fi
}

if [ "${KUBEVIRTCI_RUNTIME}" = "podman" ]; then
    _cri_socket=$(detect_podman_socket)
    _cri_bin="podman --remote --url=unix://$_cri_socket"
elif [ "${KUBEVIRTCI_RUNTIME}" = "docker" ]; then
    _cri_bin=docker
    _cri_socket="/var/run/docker.sock"
else
    _cri_socket=$(detect_podman_socket)
    if [ -n "$_cri_socket" ]; then
        _cri_bin="podman --remote --url=unix://$_cri_socket"
        >&2 echo "selecting podman as container runtime"
    elif docker ps >/dev/null 2>&1; then
        _cri_bin=docker
        _cri_socket="/var/run/docker.sock"
        >&2 echo "selecting docker as container runtime"
    else
        >&2 echo "no working container runtime found. Neither docker nor podman seems to work."
        exit 1
    fi
fi

_cli_container="${KUBEVIRTCI_GOCLI_CONTAINER:-quay.io/kubevirtci/gocli:${KUBEVIRTCI_TAG}}"
_cli="${_cri_bin} run --privileged --net=host --rm ${USE_TTY} -v ${_cri_socket}:/var/run/docker.sock"
# gocli will try to mount /lib/modules to make it accessible to dnsmasq in
# in case it exists
if [ -d /lib/modules ]; then
    _cli="${_cli} -v /lib/modules/:/lib/modules/"
fi

# Workaround https://github.com/containers/conmon/issues/315 by not dumping file content to stdout
if [[ ${_cri_bin} = podman* ]]; then
    _cli="${_cli} -v ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER:/kubevirtci_config"
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
    local params="--nodes ${KUBEVIRT_NUM_NODES} --memory ${KUBEVIRT_MEMORY_SIZE} --cpu 6 --secondary-nics ${KUBEVIRT_NUM_SECONDARY_NICS} --random-ports --background --prefix $provider_prefix ${KUBEVIRT_PROVIDER} ${KUBEVIRT_PROVIDER_EXTRA_ARGS}"

    params=" --dns-port $KUBEVIRT_DNS_HOST_PORT $params"

    if [[ $TARGET =~ windows_sysprep.* ]] && [ -n "$WINDOWS_SYSPREP_NFS_DIR" ]; then
        params=" --nfs-data $WINDOWS_SYSPREP_NFS_DIR $params"
    elif [[ $TARGET =~ windows.* ]] && [ -n "$WINDOWS_NFS_DIR" ]; then
        params=" --nfs-data $WINDOWS_NFS_DIR $params"
    elif [[ $TARGET =~ os-.* ]] && [ -n "$RHEL_NFS_DIR" ]; then
        params=" --nfs-data $RHEL_NFS_DIR $params"
    fi

    if [ -n "${KUBEVIRTCI_PROVISION_CHECK}" ]; then
        params=" --container-registry=quay.io --container-suffix=:latest $params"
    else
        if [[ -n ${KUBEVIRTCI_CONTAINER_REGISTRY} ]]; then
            params=" --container-registry=$KUBEVIRTCI_CONTAINER_REGISTRY $params"
        fi

        if [[ -n ${KUBEVIRTCI_CONTAINER_ORG} ]]; then
            params=" --container-org=$KUBEVIRTCI_CONTAINER_ORG $params"
        fi

        if [[ -n ${KUBEVIRTCI_CONTAINER_SUFFIX} ]]; then
            params=" --container-suffix=:$KUBEVIRTCI_CONTAINER_SUFFIX $params"
        fi

        if [[ ${KUBEVIRT_SLIM} == "true" ]]; then
            params=" --slim $params"
        fi
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

    if [ $KUBEVIRT_PSA == "true" ]; then
        params=" --enable-psa $params"
    fi

    if [ $KUBEVIRT_SINGLE_STACK == "true" ]; then
        params=" --single-stack $params"
    fi

    if [ $KUBEVIRT_ENABLE_AUDIT == "true" ]; then
        params=" --enable-audit $params"
    fi

    if [ $KUBEVIRT_DEPLOY_NFS_CSI == "true" ]; then
        params=" --enable-nfs-csi $params"
    fi

    # alternate (new) way to specify storage providers
    if [[ $KUBEVIRT_STORAGE == "rook-ceph-default" ]] && [[ $KUBEVIRT_PROVIDER_EXTRA_ARGS != *"--enable-ceph"* ]]; then
        params=" --enable-ceph $params"
    fi

    if [[ $KUBEVIRT_DEPLOY_PROMETHEUS == "true" ]] &&
        [[ $KUBEVIRT_PROVIDER_EXTRA_ARGS != *"--enable-prometheus"* ]]; then
        params=" --enable-prometheus $params"

        if [[ $KUBEVIRT_DEPLOY_PROMETHEUS_ALERTMANAGER == "true" ]] &&
            [[ $KUBEVIRT_PROVIDER_EXTRA_ARGS != *"--enable-prometheus-alertmanager"* ]]; then
            params=" --enable-prometheus-alertmanager $params"
        fi

        if [[ $KUBEVIRT_DEPLOY_GRAFANA == "true" ]] &&
            [[ $KUBEVIRT_PROVIDER_EXTRA_ARGS != *"--enable-grafana"* ]]; then
            params=" --enable-grafana $params"
        fi
    fi
    if [ -n "$KUBEVIRT_HUGEPAGES_2M" ]; then
        params=" --hugepages-2m $KUBEVIRT_HUGEPAGES_2M $params"
    fi

    if [ -n "$KUBEVIRT_REALTIME_SCHEDULER" ]; then
        params=" --enable-realtime-scheduler $params"
    fi

    if [ -n "$KUBEVIRT_FIPS" ]; then
        params=" --enable-fips $params"
    fi

    if [ -n "$KUBEVIRTCI_PROXY" ]; then
        params=" --docker-proxy=$KUBEVIRTCI_PROXY $params"
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
