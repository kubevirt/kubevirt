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

KUBEVIRT_WITH_KIND_ETCD_IN_MEMORY=${KUBEVIRT_WITH_KIND_ETCD_IN_MEMORY:-"true"}
ETCD_IN_MEMORY_DATA_DIR="/tmp/kind-cluster-etcd"

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
master_ip="127.0.0.1"
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

function _get_cri_bridge_mtu() {
  docker network inspect -f '{{index .Options "com.docker.network.driver.mtu"}}' bridge
}

function _patch_calico_manifest_diff() {
  local -r calico_manifest="$1"
  local -r calico_diff="$KIND_MANIFESTS_DIR/kube-calico.diff.in"

  local -r cri_mtu=$(_get_cri_bridge_mtu)
  local -r ipip_mode=$(sed -n '/name:.*CALICO_IPV4POOL_IPIP.*/{n; s/.*value:.*\(Always\|Never\).*/\1/p}' $calico_manifest)
  if [ $ipip_mode == "Always" ]; then
    overhead=$((20))
    calico_mtu=$((cri_mtu - overhead))
  else
    calico_mtu=$( sed -n 's/.*veth_mtu:.*\([[:digit:]]\{4,5\}\).*/\1/p' $calico_manifest)
  fi

  # Substitute MTU placeholder with the calculated MTU
  CNI_MTU=$calico_mtu envsubst < $calico_diff
}

function _patch_calico_manifest() {
  local -r calico_manifest="$1"
  local -r diff_string="$2"
  
  patch $calico_manifest -o - <<< "$diff_string"
}

function setup_kind() {
    $KIND --loglevel debug create cluster --retain --name=${CLUSTER_NAME} --config=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml --image=$KIND_NODE_IMAGE
    $KIND get kubeconfig --name=${CLUSTER_NAME} > ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

    docker cp ${CLUSTER_NAME}-control-plane:/kind/bin/kubectl ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl

    if [ $KUBEVIRT_WITH_KIND_ETCD_IN_MEMORY == "true" ]; then
        for node in $(_get_nodes | awk '{print $1}' | grep control-plane); do
            echo "[$node] Checking KIND cluster etcd data is mounted to RAM: $ETCD_IN_MEMORY_DATA_DIR"
            docker exec $node df -h $(dirname $ETCD_IN_MEMORY_DATA_DIR) | grep -P '(tmpfs|ramfs)'
            [ $(echo $?) != 0 ] && echo "[$node] etcd data directory is not mounted to RAM" && return 1

            docker exec $node du -h $ETCD_IN_MEMORY_DATA_DIR
            [ $(echo $?) != 0 ] && echo "[$node] Failed to check etcd data directory" && return 1
        done
    fi

    for node in $(_get_nodes | awk '{print $1}'); do
        docker exec $node /bin/sh -c "curl -L https://github.com/containernetworking/plugins/releases/download/v0.8.5/cni-plugins-linux-amd64-v0.8.5.tgz | tar xz -C /opt/cni/bin"
    done

    echo "Installing Calico CNI plugin"
    calico_manifest="$KIND_MANIFESTS_DIR/kube-calico.yaml.in"
    patched_diff=$(_patch_calico_manifest_diff $calico_manifest)
    echo "Log Calico manifest diff:"
    echo "$patched_diff"
    _patch_calico_manifest "$calico_manifest" "$patched_diff" | _kubectl apply -f -

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

function _add_worker_extra_mounts() {
    if [[ "$KUBEVIRT_PROVIDER" =~ sriov.* ]]; then
        cat <<EOF >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
  extraMounts:
  - containerPath: /lib/modules
    hostPath: /lib/modules
    readOnly: true
  - containerPath: /dev/vfio/
    hostPath: /dev/vfio/
EOF
  fi
}

function _add_worker_kubeadm_config_patch() {
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
    _add_worker_kubeadm_config_patch
    _add_worker_extra_mounts
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

function _prepare_kind_config() {
    _add_workers
    _add_kubeadm_config_patches

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
    $KIND delete cluster --name=${CLUSTER_NAME}
    rm -f ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
}
