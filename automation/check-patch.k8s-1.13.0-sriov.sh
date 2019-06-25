#!/bin/bash -e
set -x

#############################################################
# This is based on https://github.com/SchSeba/kubevirt-docker
#############################################################

export NO_PROXY="localhost,127.0.0.1,172.17.0.2"

GOPATH=~/go
GOBIN=~/go/bin
PATH=$PATH:$GOBIN

CLUSTER_NAME=sriov-ci
CLUSTER_CONTROL_PLANE=${CLUSTER_NAME}-control-plane
CONTAINER_REGISTRY_HOST="localhost:5000"

CLUSTER_CMD="docker exec -it -d ${CLUSTER_CONTROL_PLANE}"

KUBEVIRT_PATH=`pwd`
CLUSTER_DIR="cluster-up/cluster/k8s-1.13.0-sriov"
MANIFESTS_DIR="${CLUSTER_DIR}/manifests"
ARTIFACTS_DIR="${KUBEVIRT_PATH}/exported-artifacts"

# SHARED_DIR="/var/lib/stdci/shared"
# SRIOV_JOB_LOCKFILE="${SHARED_DIR}/sriov.lock"
# SRIOV_TIMEOUT_SEC="14400" # 4h

function wait_containers_ready {
    # wait until all containers are ready
    while [ -n "$(kubectl get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | grep false)" ]; do
        echo "Waiting for all containers to become ready ..."
        kubectl get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers
        sleep 10
    done
}

# NOTE: this assumes that once at least a single virt- service pops up then
# others will pop up too in quick succession, at least before the first one
# transits to ready state. If it's ever not the case, we may end up exiting
# this function before all virt pods are scheduled and in ready state. If this
# ever happens, we may need to list all services we expect in a kubevirt
# cluster and check each of them is up and running.
function wait_kubevirt_up {
    # it takes a while for virt-operator to schedule virt pods; wait for at least one of them to pop up
    while [ -z "$(kubectl get pods -n kubevirt | grep virt)" ]; do
        echo "Waiting for all pods to create ..."
        kubectl get pods -n kubevirt | grep virt
	sleep 10
    done

    wait_containers_ready
}

function collect_artifacts {
    mkdir -p "$ARTIFACTS_DIR"
    kind export logs ${ARTIFACTS_DIR} --name=${CLUSTER_NAME}
}

function finish {
    collect_artifacts
    kind delete cluster --name=${CLUSTER_NAME}
}

trap finish EXIT

tools/util/vfio.sh

# ================
# bring up cluster
# ================
# TODO FEDE takes too long to go get, move to docker image?
#go get -u sigs.k8s.io/kind 


# Create the cluster...
kind --loglevel debug create cluster --wait=$((60*60))s --retain --name=${CLUSTER_NAME} --config=${MANIFESTS_DIR}/kind.yaml --image=onesourceintegrations/node:multus

export KUBECONFIG=$(kind get kubeconfig-path --name=${CLUSTER_NAME})

kubectl cluster-info

# copied from https://github.com/kubernetes-sigs/federation-v2/blob/master/scripts/create-clusters.sh
function configure-insecure-registry-and-reload() {
    local cmd_context="${1}" # context to run command e.g. sudo, docker exec
    ${cmd_context} "$(insecure-registry-config-cmd)"
    ${cmd_context} "$(reload-docker-daemon-cmd)"
}

function reload-docker-daemon-cmd() {
    echo "kill -s SIGHUP \$(pgrep dockerd)"
}

function insecure-registry-config-cmd() {
    echo "cat <<EOF > /etc/docker/daemon.json
{
    \"insecure-registries\": [\"${CONTAINER_REGISTRY_HOST}\"]
}
EOF
"
}

configure-insecure-registry-and-reload "${CLUSTER_CMD} bash -c"

# copy config for debugging purposes
cp ${KUBECONFIG} ${CLUSTER_DIR}/cluster.config

# wait for nodes to become ready
until kubectl get nodes --no-headers
do
    echo "Waiting for all nodes to become ready ..."
    sleep 10
done

# wait until k8s pods are running
while [ -n "$(kubectl get pods --all-namespaces --no-headers | grep -v Running)" ]; do
    echo "Waiting for all pods to enter the Running state ..."
    kubectl get pods --all-namespaces --no-headers | >&2 grep -v Running || true
    sleep 10
done

# wait until all containers are ready
wait_containers_ready

# ========================
# deploy SR-IOV components
# ========================

# deploy multus
kubectl apply -f $MANIFESTS_DIR/multus.yaml

# deploy sriov cni
kubectl apply -f $MANIFESTS_DIR/sriov-crd.yaml
kubectl apply -f $MANIFESTS_DIR/sriov-cni-daemonset.yaml

# prepare kernel for vfio passthrough
modprobe vfio-pci

# deploy sriov device plugin
function configure-sriovdp() {
    local cmd_context="${1}" # context to run command e.g. sudo, docker exec
    ${cmd_context} "mkdir -p /etc/pcidp"
    ${cmd_context} "$(sriovdp-config-cmd)"
}

function sriovdp-config-cmd() {
    ./automation/configure_sriovdp.sh
    echo "cat <<EOF > /etc/pcidp/config.json
$(cat /etc/pcidp/config.json)
EOF
"
}

configure-sriovdp "${CLUSTER_CMD} bash -c"
kubectl apply -f $MANIFESTS_DIR/sriovdp-daemonset.yaml

# give them some time to create pods before checking pod status
sleep 10

# make sure all containers are ready
wait_containers_ready

# start local registry
until [ -z "$(docker ps -a | grep registry)" ]; do
    docker stop registry || true
    docker rm registry || true
    sleep 5
done
docker run -d -p 5000:5000 --restart=always --name registry registry:2
${CLUSTER_CMD} socat TCP-LISTEN:5000,fork TCP:$(docker inspect --format '{{.NetworkSettings.IPAddress }}' registry):5000

# prepare local storage
for i in {1..10}; do
    ${CLUSTER_CMD} mkdir -p /var/local/kubevirt-storage/local-volume/disk${i}
    ${CLUSTER_CMD} mkdir -p /mnt/local-storage/local/disk${i}
done
${CLUSTER_CMD} chmod -R 777 /var/local/kubevirt-storage/local-volume
${CLUSTER_CMD} mknod /dev/loop0 b 7 0

# ===============
# deploy kubevirt
# ===============
export KUBEVIRT_PROVIDER=external
export DOCKER_PREFIX=${CONTAINER_REGISTRY_HOST}/kubevirt
export DOCKER_TAG=devel
make cluster-build
make cluster-deploy
wait_kubevirt_up

# =========================
# enable sriov feature gate
# =========================
kubectl patch configmap kubevirt-config -n kubevirt --patch "data:
  feature-gates: DataVolumes, CPUManager, LiveMigration, SRIOV"

# delete all virt- pods so that they have a chance to catch up with feature gate change
kubectl get pods -n kubevirt | grep virt | awk '{print $1}' | xargs kubectl delete pods -n kubevirt
wait_kubevirt_up

# TODO FEDE Remove empty cni
${CLUSTER_CMD} rm /opt/cni/bin/sriov
docker cp /emptycni/emptycni ${CLUSTER_CONTROL_PLANE}:/opt/cni/bin/sriov

# TODO FEDE understand if there is a better way
${CLUSTER_CMD} chmod 666 /dev/vfio/vfio


# ========================
# execute functional tests
# ========================
${CLUSTER_DIR}/test.sh
