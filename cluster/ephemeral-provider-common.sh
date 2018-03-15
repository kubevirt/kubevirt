#!/bin/bash

set -e

_cli='docker run --privileged --rm -v /var/run/docker.sock:/var/run/docker.sock kubevirtci/cli@sha256:c42004c9626e6a6e723a2107410694cb80864f3456725fdf945b1ca148ed6eaa'

function _main_ip() {
    echo 127.0.0.1
}

function prepare_config() {
    BASE_PATH=${KUBEVIRT_PATH:-$PWD}
    cat >hack/config-provider-$PROVIDER.sh <<EOF
master_ip=$(_main_ip)
docker_tag=devel
kubeconfig=${BASE_PATH}/cluster/$PROVIDER/.kubeconfig
docker_prefix=localhost:5000/kubevirt
manifest_docker_prefix=registry:5000/kubevirt
EOF
}

function _registry_volume() {
  if [ -n "${JOB_NAME}" ]; then
    echo "${JOB_NAME}_${EXECUTOR_NUMBER}_registry"
  else
    echo "kubevirt_registry"
  fi
}

function build() {
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
        ${_cli} ssh --prefix $PROVIDER "node$(printf "%02d" ${i})" "echo \"${container}\" | xargs --max-args=1 sudo docker pull"
        ${_cli} ssh --prefix $PROVIDER "node$(printf "%02d" ${i})" "echo \"${container_alias}\" | xargs --max-args=2 sudo docker tag"
    done
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl "$@"
}

function down() {
    ${_cli} rm --prefix $PROVIDER
}
