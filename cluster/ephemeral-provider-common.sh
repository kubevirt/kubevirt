#!/bin/bash

set -e

_cli='docker run --privileged --rm -v /var/run/docker.sock:/var/run/docker.sock kubevirtci/cli@sha256:b0023d1863338ef04fa0b8a8ee5956ae08616200d89ffd2e230668ea3deeaff4'

function _main_ip() {
    echo 127.0.0.1
}

function _port() {
    ${_cli} port --prefix $provider_prefix "$@"
}

function prepare_config() {
    BASE_PATH=${KUBEVIRT_PATH:-$PWD}
    cat >hack/config-provider-$PROVIDER.sh <<EOF
master_ip=$(_main_ip)
docker_tag=devel
kubeconfig=${BASE_PATH}/cluster/$PROVIDER/.kubeconfig
kubectl=${BASE_PATH}/cluster/$PROVIDER/.kubectl
docker_prefix=localhost:$(_port registry)/kubevirt
manifest_docker_prefix=registry:5000/kubevirt
EOF
}

function _registry_volume() {
    echo ${job_prefix}_registry
}

function _add_common_params() {
    # Add one, 0 here means no node at all, but in the kubevirt repo it means master-only
    local num_nodes=${VAGRANT_NUM_NODES-0}
    num_nodes=$((num_nodes + 1))
    local params="--nodes ${num_nodes} --random-ports --background --prefix $provider_prefix --registry-volume $(_registry_volume) --base "kubevirtci/${image}""
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
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_TAG=${DOCKER_TAG} PROVIDER=${PROVIDER} ./hack/build-manifests.sh"
    make build docker publish

    # Make sure that all nodes use the newest images
    container=""
    container_alias=""
    for arg in ${docker_images}; do
        local name=$(basename $arg)
        container="${container} ${manifest_docker_prefix}/${name}:${docker_tag}"
        container_alias="${container_alias} ${manifest_docker_prefix}/${name}:${docker_tag} kubevirt/${name}:${docker_tag}"
    done
    local num_nodes=${VAGRANT_NUM_NODES-0}
    num_nodes=$((num_nodes + 1))
    for i in $(seq 1 ${num_nodes}); do
        ${_cli} ssh --prefix $provider_prefix "node$(printf "%02d" ${i})" "echo \"${container}\" | xargs --max-args=1 sudo docker pull"
        ${_cli} ssh --prefix $provider_prefix "node$(printf "%02d" ${i})" "echo \"${container_alias}\" | xargs --max-args=2 sudo docker tag"
    done
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl "$@"
}

function down() {
    ${_cli} rm --prefix $provider_prefix
}
