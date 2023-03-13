#!/usr/bin/env bash

set -e

# See https://github.com/k3d-io/k3d/releases
K3D_TAG=v5.4.7

function detect_cri() {
    if podman ps >/dev/null 2>&1; then echo podman; elif docker ps >/dev/null 2>&1; then echo docker; fi
}

export CRI_BIN=${CRI_BIN:-$(detect_cri)}
KUBEVIRT_NUM_SERVERS=${KUBEVIRT_NUM_SERVERS:-1}
KUBEVIRT_NUM_AGENTS=${KUBEVIRT_NUM_AGENTS:-2}

export KUBEVIRTCI_PATH
export KUBEVIRTCI_CONFIG_PATH

REGISTRY_NAME=registry
REGISTRY_HOST=127.0.0.1
KUBERNETES_SERVICE_HOST=127.0.0.1
KUBERNETES_SERVICE_PORT=6443

function _ssh_into_node() {
    ${CRI_BIN} exec -it "$@"
}

function _install_cni_plugins {
    echo "STEP: Install cnis"
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
        echo "ERROR: invalid Arch, only support x86_64, ppc64le, aarch64"
        exit 1
        ;;
    esac

    local CNI_VERSION="v0.8.5"
    local CNI_ARCHIVE="cni-plugins-linux-${ARCH}-$CNI_VERSION.tgz"
    local CNI_URL="https://github.com/containernetworking/plugins/releases/download/$CNI_VERSION/$CNI_ARCHIVE"
    if [ ! -f ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$CNI_ARCHIVE ]; then
        echo "STEP: Downloading $CNI_ARCHIVE"
        curl -sSL -o ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$CNI_ARCHIVE $CNI_URL
    fi

    for node in $(_get_nodes); do
        ${CRI_BIN} cp "${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$CNI_ARCHIVE" $node:/
        ${CRI_BIN} exec $node /bin/sh -c "mkdir -p /opt/cni/bin && tar -xvzf $CNI_ARCHIVE -C /opt/cni/bin" > /dev/null
    done
}

function _prepare_provider_config() {
    echo "STEP: Prepare provider config"
    cat >$KUBEVIRTCI_CONFIG_PATH/$KUBEVIRT_PROVIDER/config-provider-$KUBEVIRT_PROVIDER.sh <<EOF
master_ip=${KUBERNETES_SERVICE_HOST}
kubeconfig=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
kubectl=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
docker_prefix=${REGISTRY_HOST}:${HOST_PORT}/kubevirt
manifest_docker_prefix=${REGISTRY_NAME}:${HOST_PORT}/kubevirt
EOF
}

function _get_nodes() {
    _kubectl get nodes -o=custom-columns=NAME:.metadata.name --no-headers
}

function _get_agent_nodes() {
    # can be used only after _label_agents
    _kubectl get nodes -lnode-role.kubernetes.io/worker=worker -o=custom-columns=NAME:.metadata.name --no-headers
}

function _prepare_nodes {
    echo "STEP: Prepare nodes"
    for node in $(_get_nodes); do
        ${CRI_BIN} exec $node /bin/sh -c "mount --make-rshared /"
    done
}

function _install_k3d() {
    echo "STEP: Install k3d"
    curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=$K3D_TAG bash
}

function _extract_kubeconfig() {
    echo "STEP: Extract kubeconfig"
    k3d kubeconfig print $CLUSTER_NAME > ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
}

function _download_kubectl() {
    echo "STEP: Download kubectl"

    version=$(kubectl get node k3d-$CLUSTER_NAME-server-0 -o=custom-columns=VERSION:.status.nodeInfo.kubeletVersion --no-headers | cut -d + -f 1)
    curent_version=$(${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl version --short 2>/dev/null | grep Client | awk -F": " '{print $2}')

    if [[ ! -f ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl ]] || [[ $curent_version != $version ]]; then
        curl -sL https://dl.k8s.io/release/$version/bin/linux/amd64/kubectl -o ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
        chmod +x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    fi
}

function _label_agents() {
    echo "STEP: label agents"
    for node in $(_get_nodes); do
        if [[ "$node" =~ .*"agent".* ]]; then
            _kubectl label node $node node-role.kubernetes.io/worker=worker
        fi
    done
}

function _create_cluser() {
    echo "STEP: Create cluster"

    id1=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/machine-id-1
    id2=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/machine-id-2
    id3=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/machine-id-3
    printf '%.0s1' {1..32} > ${id1}
    printf '%.0s2' {1..32} > ${id2}
    printf '%.0s3' {1..32} > ${id3}

    [ $CRI_BIN == podman ] && NETWORK=podman || NETWORK=bridge

    k3d registry create --default-network $NETWORK $REGISTRY_NAME --port $REGISTRY_HOST:$HOST_PORT
    ${CRI_BIN} rename k3d-$REGISTRY_NAME $REGISTRY_NAME

    CALICO=$(pwd)/cluster-up/cluster/k3d/manifests/calico.yaml
    k3d cluster create $CLUSTER_NAME --registry-use $REGISTRY_NAME \
        --api-port $KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT \
        --servers=$KUBEVIRT_NUM_SERVERS \
        --agents=$KUBEVIRT_NUM_AGENTS \
        --k3s-arg "--disable=traefik@server:0" \
        --no-lb \
        --k3s-arg "--flannel-backend=none@server:*" \
        --k3s-arg "--kubelet-arg=cpu-manager-policy=static@agent:*" \
        --k3s-arg "--kubelet-arg=kube-reserved=cpu=500m@agent:*" \
        --k3s-arg "--kubelet-arg=system-reserved=cpu=500m@agent:*" \
        -v "$CALICO:/var/lib/rancher/k3s/server/manifests/calico.yaml@server:0" \
        -v /dev/vfio:/dev/vfio@agent:* \
        -v /lib/modules:/lib/modules@agent:* \
        -v ${id1}:/etc/machine-id@server:0 \
        -v ${id2}:/etc/machine-id@agent:0 \
        -v ${id3}:/etc/machine-id@agent:1
}

function k3d_up() {
    _install_k3d
    _create_cluser
    _extract_kubeconfig
    _download_kubectl
    _prepare_nodes
    _install_cni_plugins
    _prepare_provider_config
    _label_agents
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl --kubeconfig=$KUBECONFIG "$@"
}

function down() {
    set +e
    trap "set -e" RETURN

    for agent_node in $(_get_agent_nodes); do
        if ip netns exec $agent_node ip -details address | grep "vf 0" -B 2 > /dev/null; then
            iface=$(ip netns exec $agent_node ip -details address | grep "vf 0" -B 2 | grep "UP" | awk -F": " '{print $2}')
            ip netns exec $agent_node ip link set $iface netns 1 && echo "gracefully detached $iface from $agent_node"
        fi
    done

    ${CRI_BIN} rm --force $REGISTRY_NAME > /dev/null && echo "$REGISTRY_NAME deleted"
    k3d cluster delete $CLUSTER_NAME
}
