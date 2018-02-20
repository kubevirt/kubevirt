#!/bin/bash

function _main_ip() {
    echo 127.0.0.1
}

function up() {
    VAGRANT_NUM_NODES=${VAGRANT_NUM_NODES-0}
    # Add one, 0 here means no node at all, but in the kubevirt repo it means master-only
    VAGRANT_NUM_NODES=$((VAGRANT_NUM_NODES+1))
    docker run --privileged --rm -v /var/run/docker.sock:/var/run/docker.sock rmohr/cli:latest run --nodes ${VAGRANT_NUM_NODES} --tls-port 8443 --ssh-port 2201 --background --registry-port 5000 --base rmohr/kubeadm-1.9.3
    docker run --privileged --rm -v /var/run/docker.sock:/var/run/docker.sock rmohr/cli:latest ssh node01 sudo chown vagrant:vagrant /etc/kubernetes/admin.conf

    OPTIONS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ${KUBEVIRT_PATH}cluster/k8s-1.9.3/vagrant.key -P 2201"

    # Copy k8s config and kubectl
    scp ${OPTIONS} vagrant@127.0.0.1:/usr/bin/kubectl ${KUBEVIRT_PATH}cluster/k8s-1.9.3/.kubectl
    chmod u+x cluster/vagrant-kubernetes/.kubectl
    scp ${OPTIONS} vagrant@127.0.0.1:/etc/kubernetes/admin.conf ${KUBEVIRT_PATH}cluster/k8s-1.9.3/.kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/k8s-1.9.3/.kubeconfig
    ${KUBEVIRT_PATH}cluster/k8s-1.9.3/.kubectl config set-cluster kubernetes --server=https://127.0.0.1:8443
    ${KUBEVIRT_PATH}cluster/k8s-1.9.3/.kubectl config set-cluster kubernetes --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}

function prepare_config() {
    BASE_PATH=${KUBEVIRT_PATH:-$PWD}
    cat >hack/config-provider-k8s-1.9.3.sh <<EOF
master_ip=$(_main_ip)
docker_tag=devel
kubeconfig=${BASE_PATH}/cluster/k8s-1.9.3/.kubeconfig
docker_prefix=localhost:5000/kubevirt
manifest_docker_prefix=registry:5000/kubevirt
EOF
}

function build() {
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_TAG=${DOCKER_TAG} PROVIDER=${PROVIDER} ./hack/build-manifests.sh"
    make build docker publish
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/k8s-1.9.3/.kubeconfig
    ${KUBEVIRT_PATH}cluster/k8s-1.9.3/.kubectl "$@"
}

function down() {
    docker run --privileged --rm -v /var/run/docker.sock:/var/run/docker.sock rmohr/cli:latest rm
}
