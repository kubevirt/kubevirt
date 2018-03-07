#!/bin/bash

set -e

prefix=kubevirt-os-3.9.0-alpha.4

function _main_ip() {
    echo 127.0.0.1
}

_cli='docker run --privileged --rm -v /var/run/docker.sock:/var/run/docker.sock rmohr/cli:latest'

function up() {
    # Add one, 0 here means no node at all, but in the kubevirt repo it means master-only
    local num_nodes=${VAGRANT_NUM_NODES-0}
    num_nodes=$((num_nodes + 1))
    ${_cli} run --nodes ${num_nodes} --osp-port 127.0.0.1:8443 --ssh-port 127.0.0.1:2201 --background --registry-port 127.0.0.1:5000 --prefix $prefix --registry-volume kubevirt_registry --base "rmohr/os-3.9"

    chmod 0600 ${KUBEVIRT_PATH}cluster/vagrant.key
    OPTIONS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ${KUBEVIRT_PATH}cluster/vagrant.key -P 2201"

    # Copy oc tool
    scp ${OPTIONS} vagrant@127.0.0.1:/usr/local/bin/oc ${KUBEVIRT_PATH}cluster/os-3.9.0-alpha.4/.oc
    chmod u+x ${KUBEVIRT_PATH}cluster/os-3.9.0-alpha.4/.oc

    # Login to OpenShift
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/os-3.9.0-alpha.4/.kubeconfig
    ${KUBEVIRT_PATH}cluster/os-3.9.0-alpha.4/.oc login $(_main_ip):8443 --insecure-skip-tls-verify=true -u admin -p admin

    # Make sure that local config is correct
    prepare_config
}

function prepare_config() {
    BASE_PATH=${KUBEVIRT_PATH:-$PWD}
    cat >hack/config-provider-os-3.9.0-alpha.4.sh <<EOF
master_ip=$(_main_ip)
docker_tag=devel
kubeconfig=${BASE_PATH}/cluster/os-3.9.0-alpha.4/.kubeconfig
docker_prefix=localhost:5000/kubevirt
manifest_docker_prefix=registry:5000/kubevirt
EOF
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
        ${_cli} ssh --prefix $prefix "node$(printf "%02d" ${i})" "echo \"${container}\" | xargs --max-args=1 sudo docker pull"
        ${_cli} ssh --prefix $prefix "node$(printf "%02d" ${i})" "echo \"${container_alias}\" | xargs --max-args=2 sudo docker tag"
    done
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/os-3.9.0-alpha.4/.kubeconfig
    ${KUBEVIRT_PATH}cluster/os-3.9.0-alpha.4/.oc "$@"
}

function down() {
    ${_cli} rm --prefix $prefix
}
