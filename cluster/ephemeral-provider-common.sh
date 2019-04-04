#!/usr/bin/env bash

set -e

_cli_container="kubevirtci/gocli@sha256:4a4565eb4487be95f464cb942590bbaa980a5b56acb32d955f7a9f81f4ba843c"
_cli_with_tty="docker run --privileged --net=host --rm -t -v /var/run/docker.sock:/var/run/docker.sock ${_cli_container}"
_cli="docker run --privileged --net=host --rm ${USE_TTY} -v /var/run/docker.sock:/var/run/docker.sock ${_cli_container}"

function _main_ip() {
    echo 127.0.0.1
}

function _port() {
    ${_cli} ports --prefix $provider_prefix "$@"
}

function prepare_config() {
    BASE_PATH=${KUBEVIRT_PATH:-$PWD}
    cat >hack/config-provider-$KUBEVIRT_PROVIDER.sh <<EOF
master_ip=$(_main_ip)
docker_tag=devel
docker_tag_alt=devel_alt
kubeconfig=${BASE_PATH}/cluster/$KUBEVIRT_PROVIDER/.kubeconfig
kubectl=${BASE_PATH}/cluster/$KUBEVIRT_PROVIDER/.kubectl
docker_prefix=localhost:$(_port registry)/kubevirt
manifest_docker_prefix=registry:5000/kubevirt
EOF
}

function _registry_volume() {
    echo ${job_prefix}_registry
}

function _add_common_params() {
    local params="--nodes ${KUBEVIRT_NUM_NODES} --memory ${KUBEVIRT_MEMORY_SIZE} --cpu 5 --random-ports --background --prefix $provider_prefix --registry-volume $(_registry_volume) kubevirtci/${image} ${KUBEVIRT_PROVIDER_EXTRA_ARGS}"
    if [[ $TARGET =~ windows.* ]]; then
        params=" --nfs-data $WINDOWS_NFS_DIR $params"
    elif [[ $TARGET =~ os-.* ]]; then
        params=" --nfs-data $RHEL_NFS_DIR $params"
    fi
    echo $params
}

function build() {
    # Build everyting and publish it
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/build-manifests.sh"
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-build.sh"
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-push-images.sh"

    # Make sure that all nodes use the newest images
    container=""
    container_alias=""
    for arg in ${docker_images}; do
        local name=$(basename $arg)
        container="${container} ${manifest_docker_prefix}/${name}:${docker_tag} ${manifest_docker_prefix}/${name}:${docker_tag_alt}"
        container_alias="${container_alias} ${manifest_docker_prefix}/${name}:${docker_tag} kubevirt/${name}:${docker_tag}"
    done
    for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
        ${_cli_with_tty} ssh --prefix $provider_prefix "node$(printf "%02d" ${i})" "echo \"${container}\" | xargs \-\-max-args=1 sudo docker pull"
        ${_cli_with_tty} ssh --prefix $provider_prefix "node$(printf "%02d" ${i})" "echo \"${container_alias}\" | xargs \-\-max-args=2 sudo docker tag"
    done
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubectl "$@"
}

function down() {
    ${_cli} rm --prefix $provider_prefix
}
