#!/bin/bash

set -e

_cli="docker run --privileged --net=host --rm ${USE_TTY} -v /var/run/docker.sock:/var/run/docker.sock kubevirtci/gocli@sha256:aa7f295a7908fa333ab5e98ef3af0bfafbabfd3cee2b83f9af47f722e3000f6a"

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
    local params="--nodes ${KUBEVIRT_NUM_NODES} --random-ports --background --prefix $provider_prefix --registry-volume $(_registry_volume) kubevirtci/${image} ${KUBEVIRT_PROVIDER_EXTRA_ARGS}"
    if [ -d "$NFS_WINDOWS_DIR" ]; then
        params="--memory 8192M --nfs-data $NFS_WINDOWS_DIR $params"
    fi
    echo $params
}

function build() {
    # Let's first prune old images, keep the last 5 iterations to improve the cache hit chance
    for arg in ${docker_images}; do
        local name=$(basename $arg)
        images_to_prune="$(docker images --filter "label=${job_prefix}" --filter "label=${name}" --format="{{.ID}} {{.Repository}}:{{.Tag}}" | cat -n | sort -uk2,2 | sort -k1 | tr -s ' ' | grep -v "<none>" | cut -d' ' -f3 | tail -n +6)"
        if [ -n "${images_to_prune}" ]; then
            docker rmi ${images_to_prune}
        fi
    done

    # Build everyting and publish it
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/build-manifests.sh"
    make build docker publish

    # Make sure that all nodes use the newest images
    container=""
    container_alias=""
    for arg in ${docker_images}; do
        local name=$(basename $arg)
        container="${container} ${manifest_docker_prefix}/${name}:${docker_tag}"
        container_alias="${container_alias} ${manifest_docker_prefix}/${name}:${docker_tag} kubevirt/${name}:${docker_tag}"
    done
    for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
        ${_cli} ssh --prefix $provider_prefix "node$(printf "%02d" ${i})" "echo \"${container}\" | xargs \-\-max-args=1 sudo docker pull"
        ${_cli} ssh --prefix $provider_prefix "node$(printf "%02d" ${i})" "echo \"${container_alias}\" | xargs \-\-max-args=2 sudo docker tag"
    done
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubectl "$@"
}

function down() {
    ${_cli} rm --prefix $provider_prefix
}
