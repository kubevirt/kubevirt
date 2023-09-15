#!/usr/bin/env bash

set -e

function detect_cri() {
    if podman ps >/dev/null 2>&1; then echo podman; elif docker ps >/dev/null 2>&1; then echo docker; fi
}

export CRI_BIN=${CRI_BIN:-$(detect_cri)}
CONFIG_WORKER_CPU_MANAGER=${CONFIG_WORKER_CPU_MANAGER:-false}
# only setup ipFamily when the environmental variable is not empty
# avaliable value: ipv4, ipv6, dual
IPFAMILY=${IPFAMILY}

# check CPU arch
PLATFORM=$(uname -m)
case ${PLATFORM} in
x86_64* | i?86_64* | amd64*)
    ARCH="amd64"
    ;;
ppc64le)
    ARCH="ppc64le"
    ;;
aarch64* | arm64*)
    ARCH="arm64"
    ;;
*)
    echo "invalid Arch, only support x86_64, ppc64le, aarch64"
    exit 1
    ;;
esac

NODE_CMD="${CRI_BIN} exec -it -d "
export KIND_MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/kind/manifests"
export KIND_NODE_CLI="${CRI_BIN} exec -it "
export KUBEVIRTCI_PATH
export KUBEVIRTCI_CONFIG_PATH
KIND_DEFAULT_NETWORK="kind"

KUBECTL="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl --kubeconfig=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig"

REGISTRY_NAME=${CLUSTER_NAME}-registry

MASTER_NODES_PATTERN="control-plane"
WORKER_NODES_PATTERN="worker"

KUBEVIRT_WITH_KIND_ETCD_IN_MEMORY=${KUBEVIRT_WITH_KIND_ETCD_IN_MEMORY:-"true"}
ETCD_IN_MEMORY_DATA_DIR="/tmp/kind-cluster-etcd"

function _wait_kind_up {
    echo "Waiting for kind to be ready ..."
    if [[ $KUBEVIRT_PROVIDER =~ kind-.*1\.1.* ]]; then
        selector="master"
    else
        selector="control-plane"
    fi
    while [ -z "$(${CRI_BIN} exec --privileged ${CLUSTER_NAME}-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf get nodes --selector=node-role.kubernetes.io/${selector} -o=jsonpath='{.items..status.conditions[-1:].status}' | grep True)" ]; do
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
    KIND="${KUBEVIRTCI_CONFIG_PATH}"/"$KUBEVIRT_PROVIDER"/.kind
    current_kind_version=$($KIND --version |& awk '{print $3}')
    if [[ $current_kind_version != $KIND_VERSION ]]; then
        echo "Downloading kind v$KIND_VERSION"
        curl -LSs https://github.com/kubernetes-sigs/kind/releases/download/v$KIND_VERSION/kind-linux-${ARCH} -o "$KIND"
        chmod +x "$KIND"
    fi
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
    ${CRI_BIN} exec -it "$1" bash
}

function _run_registry() {
    local -r network=${1}

    until [ -z "$($CRI_BIN ps -a | grep $REGISTRY_NAME)" ]; do
        ${CRI_BIN} stop $REGISTRY_NAME || true
        ${CRI_BIN} rm $REGISTRY_NAME || true
        sleep 5
    done
    ${CRI_BIN} run -d --network=${network} -p $HOST_PORT:5000  --restart=always --name $REGISTRY_NAME quay.io/kubevirtci/library-registry:2.7.1

}

function _configure_registry_on_node() {
    local -r node=${1}
    local -r network=${2}

    _configure-insecure-registry-and-reload "${NODE_CMD} ${node} bash -c"
    ${NODE_CMD} ${node} sh -c "echo $(${CRI_BIN} inspect --format "{{.NetworkSettings.Networks.${network}.IPAddress }}" $REGISTRY_NAME)'\t'registry >> /etc/hosts"
}

function _install_cnis {
    _install_cni_plugins
}

function _install_cni_plugins {
    local CNI_VERSION="v0.8.5"
    local CNI_ARCHIVE="cni-plugins-linux-${ARCH}-$CNI_VERSION.tgz"
    local CNI_URL="https://github.com/containernetworking/plugins/releases/download/$CNI_VERSION/$CNI_ARCHIVE"
    if [ ! -f ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$CNI_ARCHIVE ]; then
        echo "Downloading $CNI_ARCHIVE"
        curl -sSL -o ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$CNI_ARCHIVE $CNI_URL
    fi

    for node in $(_get_nodes | awk '{print $1}'); do
        ${CRI_BIN} cp "${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$CNI_ARCHIVE" $node:/
        ${CRI_BIN} exec $node /bin/sh -c "tar xf $CNI_ARCHIVE -C /opt/cni/bin"
    done
}

function prepare_config() {
    BASE_PATH=${KUBEVIRTCI_CONFIG_PATH:-$PWD}
    cat >$BASE_PATH/$KUBEVIRT_PROVIDER/config-provider-$KUBEVIRT_PROVIDER.sh <<EOF
master_ip="127.0.0.1"
kubeconfig=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
kubectl=${BASE_PATH}/$KUBEVIRT_PROVIDER/.kubectl
docker_prefix=localhost:${HOST_PORT}/kubevirt
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

function _get_nodes() {
    _kubectl get nodes --no-headers
}

function _get_pods() {
    _kubectl get pods --all-namespaces --no-headers
}

function _fix_node_labels() {
    # Due to inconsistent labels and taints state in multi-nodes clusters,
    # it is nessecery to remove taint NoSchedule and set role labels manualy:
    #   Control-plane nodes might lack 'scheduable=true' label and have NoScheduable taint.
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

    ${CRI_BIN} cp ${CLUSTER_NAME}-control-plane:$KUBECTL_PATH ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl

    if [ $KUBEVIRT_WITH_KIND_ETCD_IN_MEMORY == "true" ]; then
        for node in $(_get_nodes | awk '{print $1}' | grep control-plane); do
            echo "[$node] Checking KIND cluster etcd data is mounted to RAM: $ETCD_IN_MEMORY_DATA_DIR"
            ${CRI_BIN} exec $node df -h $(dirname $ETCD_IN_MEMORY_DATA_DIR) | grep -P '(tmpfs|ramfs)'
            [ $(echo $?) != 0 ] && echo "[$node] etcd data directory is not mounted to RAM" && return 1

            ${CRI_BIN} exec $node du -h $ETCD_IN_MEMORY_DATA_DIR
            [ $(echo $?) != 0 ] && echo "[$node] Failed to check etcd data directory" && return 1
        done
    fi

    _install_cnis

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
    _run_registry "$KIND_DEFAULT_NETWORK"

    for node in $(_get_nodes | awk '{print $1}'); do
        _configure_registry_on_node "$node" "$KIND_DEFAULT_NETWORK"
        _configure_network "$node"
    done
    prepare_config
}

function _add_extra_mounts() {
  cat <<EOF >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
  extraMounts:
  - containerPath: /var/log/audit
    hostPath: /var/log/audit
    readOnly: true
EOF

    if [[ "$KUBEVIRT_PROVIDER" =~ sriov.* || "$KUBEVIRT_PROVIDER" =~ vgpu.* ]]; then
        cat <<EOF >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
  - containerPath: /dev/vfio/
    hostPath: /dev/vfio/
EOF
  fi
}

function _add_kubeadm_cpu_manager_config_patch() {
    cat << EOF >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
  kubeadmConfigPatches:
  - |-
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        "feature-gates": "CPUManager=true"
        "cpu-manager-policy": "static"
        "kube-reserved": "cpu=500m"
        "system-reserved": "cpu=500m"
EOF
}

function _add_workers() {
    # appending eventual workers to the yaml
    for ((n=0;n<$(($KUBEVIRT_NUM_NODES-1));n++)); do
        cat << EOF >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
- role: worker
EOF
    if [ $CONFIG_WORKER_CPU_MANAGER == true ]; then
         _add_kubeadm_cpu_manager_config_patch
    fi
    _add_extra_mounts
    done
}

function _add_kubeadm_config_patches() {
    if [ $KUBEVIRT_WITH_KIND_ETCD_IN_MEMORY == "true" ]; then
        cat <<EOF >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
kubeadmConfigPatches:
- |
  kind: ClusterConfiguration
  metadata:
    name: config
  etcd:
    local:
      dataDir: $ETCD_IN_MEMORY_DATA_DIR
EOF
        echo "KIND cluster etcd data will be mounted to RAM on kind nodes: $ETCD_IN_MEMORY_DATA_DIR"
    fi
}

function _setup_ipfamily() {
    if [ $IPFAMILY != "" ]; then
        cat <<EOF >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
networking:
  ipFamily: $IPFAMILY
EOF
        echo "KIND cluster ip family has been set to $IPFAMILY"
    fi
}

function _prepare_kind_config() {
    _add_workers
    _add_kubeadm_config_patches
    _setup_ipfamily
    echo "Final KIND config:"
    cat ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
}

function kind_up() {
    _fetch_kind
    _prepare_kind_config
    setup_kind
}

function _kubectl() {
    ${KUBECTL} "$@"
}

function down() {
    _fetch_kind
    if [ -z "$($KIND get clusters | grep ${CLUSTER_NAME})" ]; then
        return
    fi

    worker_nodes=$(_get_nodes | grep -i $WORKER_NODES_PATTERN | awk '{print $1}')
    for worker_node in $worker_nodes; do
        if ip netns exec $worker_node ip -details address | grep "vf 0" -B 2 > /dev/null; then
            iface=$(ip netns exec $worker_node ip -details address | grep "vf 0" -B 2 | grep -E 'UP|DOWN' | awk -F": " '{print $2}')
            ip netns exec $worker_node ip link set $iface netns 1 && echo "gracefully detached $iface from $worker_node"
        fi
    done

    # On CI, avoid failing an entire test run just because of a deletion error
    $KIND delete cluster --name=${CLUSTER_NAME} || [ "$CI" = "true" ]
    ${CRI_BIN} rm -f $REGISTRY_NAME >> /dev/null
    rm -f ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
}
