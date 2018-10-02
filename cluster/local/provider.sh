#!/bin/bash

function _cert_dir() {
    echo $GOPATH/src/kubevirt.io/kubevirt/cluster/local/certs
}

function _main_ip() {
    ip -o -4 a | tr -s ' ' | cut -d' ' -f 2,4 |
        grep -v -e '^lo[0-9:]*' | head -1 |
        cut -d' ' -f 2 | cut -d'/' -f1
}

function up() {
    # Make sure that local config is correct
    prepare_config

    go get -d k8s.io/kubernetes

    export API_HOST_IP=$(_main_ip)
    export KUBELET_HOST=$(_main_ip)
    export HOSTNAME_OVERRIDE=kubdev
    export ALLOW_PRIVILEGED=1
    export ALLOW_SECURITY_CONTEXT=1
    export KUBE_DNS_DOMAIN="cluster.local"
    export KUBE_DNS_SERVER_IP="10.0.0.10"
    export KUBE_ENABLE_CLUSTER_DNS=true
    export CERT_DIR=$(_cert_dir)
    (
        cd $GOPATH/src/k8s.io/kubernetes
        ./hack/local-up-cluster.sh
    )
}

function prepare_config() {
    cat >hack/config-provider-local.sh <<EOF
master_ip=$(_main_ip)
docker_tag=devel
kubeconfig=$(_cert_dir)/admin.kubeconfig
EOF
}

function build() {
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_TAG=${DOCKER_TAG}
    KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/build-manifests.sh"
    make docker
}

function _kubectl() {
    export KUBECONFIG=$(_cert_dir)/admin.kubeconfig
    $GOPATH/src/k8s.io/kubernetes/cluster/kubectl.sh "$@"
}

function down() {
    echo "Not supported by this provider. Please kill the running script manually."
}
