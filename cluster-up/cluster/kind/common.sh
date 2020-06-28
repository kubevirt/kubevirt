#!/usr/bin/env bash

set -e

NODE_CMD="docker exec -it -d "
export KIND_MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/kind/manifests"
export KIND_NODE_CLI="docker exec -it "
export KUBEVIRTCI_PATH
export KUBEVIRTCI_CONFIG_PATH

KUBECTL="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl --kubeconfig=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig"

REGISTRY_NAME=${CLUSTER_NAME}-registry

MASTER_NODES_PATTERN="control-plane"
WORKER_NODES_PATTERN="worker"

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
        wget https://github.com/kubernetes-sigs/kind/releases/download/v0.7.0/kind-linux-amd64 -O ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kind
        chmod +x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kind
    fi
    KIND=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kind
}

function _configure-insecure-registry-and-reload() {
    local cmd_context="${1}" # context to run command e.g. sudo, docker exec
    ${cmd_context} "$(_insecure-registry-config-cmd)"
    ${cmd_context} "$(_reload-containerd-daemon-cmd)"
}

function _reload-containerd-daemon-cmd() {
    echo "systemctl restart containerd"
}

function _insecure-registry-config-cmd() {
    echo "sed -i '/\[plugins.cri.registry.mirrors\]/a\        [plugins.cri.registry.mirrors.\"registry:5000\"]\n\          endpoint = [\"http://registry:5000\"]' /etc/containerd/config.toml"
}

# this works since the nodes use the same names as containers
function _ssh_into_node() {
    docker exec -it "$1" bash
}

function _run_registry() {
    until [ -z "$(docker ps -a | grep $REGISTRY_NAME)" ]; do
        docker stop $REGISTRY_NAME || true
        docker rm $REGISTRY_NAME || true
        sleep 5
    done
    docker run -d -p 5000:5000 --restart=always --name $REGISTRY_NAME registry:2
}

function _configure_registry_on_node() {
    _configure-insecure-registry-and-reload "${NODE_CMD} $1 bash -c"
    ${NODE_CMD} $1  sh -c "echo $(docker inspect --format '{{.NetworkSettings.IPAddress }}' $REGISTRY_NAME)'\t'registry >> /etc/hosts"
}

function prepare_config() {
    BASE_PATH=${KUBEVIRTCI_CONFIG_PATH:-$PWD}
    cat >$BASE_PATH/$KUBEVIRT_PROVIDER/config-provider-$KUBEVIRT_PROVIDER.sh <<EOF
if [ -z ${IPV6_CNI+x} ]; then
    master_ip="127.0.0.1"
else
    master_ip="::1"
fi
kubeconfig=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
kubectl=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubectl
docker_prefix=localhost:5000/kubevirt
manifest_docker_prefix=registry:5000/kubevirt
EOF
}

function _configure_network() {
    # modprobe is present inside kind container but may be missing in the
    # environment running this script, so load the module from inside kind
    ${NODE_CMD} $1 modprobe br_netfilter
    for knob in arp ip ip6; do
        ${NODE_CMD} $1 sysctl -w sys.net.bridge.bridge-nf-call-${knob}tables=1
    done
}

function prepare_workers() {
    # appending eventual workers to the yaml
    for ((n=0;n<$(($KUBEVIRT_NUM_NODES-1));n++)); do
        cat << EOF >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
- role: worker
  kubeadmConfigPatches:
  - |
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        "feature-gates": "CPUManager=true"
        "cpu-manager-policy": "static"
        "kube-reserved": "cpu=500m"
        "system-reserved": "cpu=500m"
EOF
    done
}

function _get_nodes() {
    _kubectl get nodes --no-headers
}

function _get_pods() {
    _kubectl get pods --all-namespaces --no-headers
}

function _fix_node_labels() {
    # Due to inconsistent labels and taints state in multi-nodes clusters,
    # it is nessecery to remove taint NoSchedule and set role labels manualy:
    #   Master nodes might lack 'scheduable=true' label and have NoScheduable taint.
    #   Worker nodes might lack worker role label.
    master_nodes=$(_get_nodes | grep -i $MASTER_NODES_PATTERN | awk '{print $1}')
    for node in ${master_nodes[@]}; do
        # removing NoSchedule taint if is there
        if _kubectl taint nodes $node node-role.kubernetes.io/master:NoSchedule-; then
            _kubectl label node $node kubevirt.io/schedulable=true
        fi
    done

    worker_nodes=$(_get_nodes | grep -i $WORKER_NODES_PATTERN | awk '{print $1}')
    for node in ${worker_nodes[@]}; do
        _kubectl label node $node kubevirt.io/schedulable=true
        _kubectl label node $node node-role.kubernetes.io/worker=""
    done
}

function setup_kind() {
    $KIND --loglevel debug create cluster --retain --name=${CLUSTER_NAME} --config=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml --image=$KIND_NODE_IMAGE
    $KIND get kubeconfig --name=${CLUSTER_NAME} > ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

    docker cp ${CLUSTER_NAME}-control-plane:/kind/bin/kubectl ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl

    for node in $(_get_nodes | awk '{print $1}'); do
        docker exec $node /bin/sh -c "curl -L https://github.com/containernetworking/plugins/releases/download/v0.8.5/cni-plugins-linux-amd64-v0.8.5.tgz | tar xz -C /opt/cni/bin"
    done

    echo "ipv6 cni: $IPV6_CNI"
    if [ -z ${IPV6_CNI+x} ]; then
        echo "no ipv6, safe to install calico"
        _kubectl apply -f $KIND_MANIFESTS_DIR/kube-calico.yaml
    else
        echo "ipv6 enabled, using kindnet"
        # currently kind does not fully support ipv6 or ipv6-DualStack,
        # when using diffrent CNI's.
    fi

    _wait_kind_up
    _kubectl cluster-info
    _fix_node_labels

    until _get_nodes
    do
        echo "Waiting for all nodes to become ready ..."
        sleep 10
    done

    # wait until k8s pods are running
    while [ -n "$(_get_pods | grep -v Running)" ]; do
        echo "Waiting for all pods to enter the Running state ..."
        _get_pods | >&2 grep -v Running || true
        sleep 10
    done

    _wait_containers_ready
    _run_registry

    for node in $(_get_nodes | awk '{print $1}'); do
        _configure_registry_on_node "$node"
        _configure_network "$node"
    done
    prepare_config
}

function kind_up() {
    _fetch_kind
    prepare_workers
    setup_kind
}

function _kubectl() {
    ${KUBECTL} "$@"
}

function down() {
    _fetch_kind
    if [ -z $($KIND get clusters | grep ${CLUSTER_NAME}) ]; then
        return
    fi
    $KIND delete cluster --name=${CLUSTER_NAME}
    rm ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
}
