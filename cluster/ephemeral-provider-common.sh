#!/bin/bash

set -e

_cli="docker run --privileged --net=host --rm ${USE_TTY} -v /var/run/docker.sock:/var/run/docker.sock kubevirtci/gocli@sha256:4a4565eb4487be95f464cb942590bbaa980a5b56acb32d955f7a9f81f4ba843c"

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
    local params="--nodes ${KUBEVIRT_NUM_NODES} --memory ${KUBEVIRT_MEMORY_SIZE} --cpu 5 --random-ports --background --prefix $provider_prefix --registry-volume $(_registry_volume) kubevirtci/${image} ${KUBEVIRT_PROVIDER_EXTRA_ARGS}"
    if [[ $TARGET =~ windows.* ]]; then
        params=" --nfs-data $WINDOWS_NFS_DIR $params"
    elif [[ $TARGET =~ openshift.* ]]; then
        params=" --nfs-data $RHEL_NFS_DIR $params"
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
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/build-manifests.sh"
    make push

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
