#!/usr/bin/env bash

set -e

NODE_CMD="docker exec -it -d "
export KIND_MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/kind/manifests"
export KIND_NODE_CLI="docker exec -it "
export KUBEVIRTCI_PATH

function _wait_kind_up {
    echo "Waiting for kind to be ready ..."  
    while [ -z "$(docker exec --privileged ${CLUSTER_NAME}-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf get nodes --selector=node-role.kubernetes.io/master -o=jsonpath='{.items..status.conditions[-1:].status}' | grep True)" ]; do
        echo "Waiting for kind to be ready ..."        
        sleep 10
    done
    echo "Waiting for dns to be ready ..."        
    _kubectl wait -n kube-system --timeout=12m --for=condition=Ready -l k8s-app=kube-dns pods
}

function _wait_containers_ready {
    echo "Waiting for all containers to become ready ..."
    _kubectl wait --for=condition=Ready pod --all -n kube-system --timeout 12m
}

function _fetch_kind() {
    if [ ! -f ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kind ]; then
        wget https://github.com/kubernetes-sigs/kind/releases/download/v0.3.0/kind-linux-amd64 -O ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kind
        chmod +x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kind
    fi
    KIND=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kind
}

function _configure-insecure-registry-and-reload() {
    local cmd_context="${1}" # context to run command e.g. sudo, docker exec
    ${cmd_context} "$(_insecure-registry-config-cmd)"
    ${cmd_context} "$(_reload-docker-daemon-cmd)"
}

function _reload-docker-daemon-cmd() {
    echo "kill -s SIGHUP \$(pgrep dockerd)"
}

function _insecure-registry-config-cmd() {
    echo "cat <<EOF > /etc/docker/daemon.json
{
    \"insecure-registries\": [\"${CONTAINER_REGISTRY_HOST}\"]
}
EOF
"
}

# this works since the nodes use the same names as containers
function _ssh_into_node() {
    docker exec -it "$1" bash
}

function _run_registry() {
    until [ -z "$(docker ps -a | grep registry)" ]; do
        docker stop registry || true
        docker rm registry || true
        sleep 5
    done
    docker run -d -p 5000:5000 --restart=always --name registry registry:2
}

function _configure_registry_on_node() {
    _configure-insecure-registry-and-reload "${NODE_CMD} $1 bash -c"
    ${NODE_CMD} $1 socat TCP-LISTEN:5000,fork TCP:$(docker inspect --format '{{.NetworkSettings.IPAddress }}' registry):5000
}

function prepare_config() {
    BASE_PATH=${KUBEVIRTCI_CONFIG_PATH:-$PWD}
    cat >$BASE_PATH/$KUBEVIRT_PROVIDER/config-provider-$KUBEVIRT_PROVIDER.sh <<EOF
master_ip="127.0.0.1"
kubeconfig=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
kubectl=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubectl
docker_prefix=localhost:5000/kubevirt
manifest_docker_prefix=localhost:5000/kubevirt
EOF
}

function kind_up() {
    _fetch_kind
  
    # appending eventual workers to the yaml
    for ((n=0;n<$(($KUBEVIRT_NUM_NODES-1));n++)); do 
        echo "- role: worker" >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
    done

    $KIND --loglevel debug create cluster --retain --name=${CLUSTER_NAME} --config=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml --image=$KIND_NODE_IMAGE
    cp $($KIND get kubeconfig-path --name=${CLUSTER_NAME}) ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

    docker cp ${CLUSTER_NAME}-control-plane:/kind/bin/kubectl ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    
    _kubectl create -f $KIND_MANIFESTS_DIR/kube-flannel.yaml

    _wait_kind_up
    _kubectl cluster-info

    until _kubectl get nodes --no-headers
    do
        echo "Waiting for all nodes to become ready ..."
        sleep 10
    done

    # wait until k8s pods are running
    while [ -n "$(_kubectl get pods --all-namespaces --no-headers | grep -v Running)" ]; do
        echo "Waiting for all pods to enter the Running state ..."
        _kubectl get pods --all-namespaces --no-headers | >&2 grep -v Running || true
        sleep 10
    done

    _wait_containers_ready
    _run_registry

    for node in $(_kubectl get nodes --no-headers | awk '{print $1}'); do
        _configure_registry_on_node "$node"
    done
    prepare_config
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl "$@"
}

function down() {
    _fetch_kind
    if [ -z $($KIND get clusters | grep ${CLUSTER_NAME}) ]; then
        return
    fi
    $KIND delete cluster --name=${CLUSTER_NAME}
    rm ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
}
