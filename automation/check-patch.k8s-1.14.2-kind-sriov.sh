#!/bin/bash -e
set -x

#############################################################
# CI Notes TODO Move in a more appropriate location
# Each CI node labeled as sriov-enable should have the following setup in place:
# - have the vfio_pci module loaded with a modprobe.d rule
# - since we use kind, in order to have the dns working we need to
#   - have the br_netfilter module loaded with a modprobe.d rule
#   - have the following lines on /etc/sysctl.conf (see https://stackoverflow.com/questions/48148838/kube-dns-error-reply-from-unexpected-source)
#               net.bridge.bridge-nf-call-ip6tables = 1
#               net.bridge.bridge-nf-call-iptables = 1
#               net.bridge.bridge-nf-call-arptables = 1
#
#############################################################


#############################################################
# This is based on https://github.com/SchSeba/kubevirt-docker
#############################################################

export NO_PROXY="localhost,127.0.0.1,172.17.0.2"

export WORKSPACE="${WORKSPACE:-$PWD}"
readonly ARTIFACTS_PATH="${ARTIFACTS-$WORKSPACE/exported-artifacts}"

CLUSTER_NAME=sriov-ci
CLUSTER_CONTROL_PLANE=${CLUSTER_NAME}-control-plane
CONTAINER_REGISTRY_HOST="localhost:5000"

CLUSTER_CMD="docker exec -it -d ${CLUSTER_CONTROL_PLANE}"

KUBEVIRT_PATH=`pwd`
CLUSTER_DIR="cluster-up/cluster/k8s-1.14.2-kind-sriov"
MANIFESTS_DIR="${CLUSTER_DIR}/manifests"


function wait_containers_ready {
    echo "Waiting for all containers to become ready ..."
    kubectl wait --for=condition=Ready pod --all -n kube-system --timeout 12m
}

function wait_kubevirt_up {
    echo "Waiting for kubevirt to be ready ..."
    kubectl wait --for=condition=Ready pod --all -n kubevirt --timeout 12m
    kubectl wait --for=condition=Ready pod --all -n cdi --timeout 12m
}

function wait_kind_up {
    while [ -z "$(docker exec --privileged ${CLUSTER_CONTROL_PLANE} kubectl --kubeconfig=/etc/kubernetes/admin.conf get nodes --selector=node-role.kubernetes.io/master -o=jsonpath='{.items..status.conditions[-1:].status}' | grep True)" ]; do
        echo "Waiting for kind to be ready ..."        
	    sleep 10
    done
}

function collect_artifacts {
    kind export logs ${ARTIFACTS_PATH} --name=${CLUSTER_NAME} || true
}

function finish {
    collect_artifacts
    kind delete cluster --name=${CLUSTER_NAME}
}

function enable_vfio {
    counter=0
    for file in $(find /sys/devices/ -name *sriov_totalvfs*); do
        pfroot=$(dirname $file)

        # enable all enabled VFs. If it fails means that sysfs is not supported on that device and we pass
        cat $file > $pfroot/sriov_numvfs || continue 

        # bind all VFs with vfio
        for virtfn in $(ls -d $pfroot/virtfn*); do
            pciid=$(basename $(readlink $virtfn))
            if [ -e $virtfn/driver/unbind ]; then
                echo $pciid > $virtfn/driver/unbind
            fi
            echo $(lspci -n -s $pciid | sed 's/:/ /g' | awk '{print $4 " " $5}') > /sys/bus/pci/drivers/vfio-pci/new_id
            counter=$((counter+1))
        done
    done
}

trap finish EXIT

enable_vfio

# Create the cluster...
wget https://github.com/kubernetes-sigs/kind/releases/download/v0.3.0/kind-linux-amd64 -O /usr/local/bin/kind
chmod +x /usr/local/bin/kind
kind --loglevel debug create cluster --retain --name=${CLUSTER_NAME} --config=${MANIFESTS_DIR}/kind.yaml
export KUBECONFIG=$(kind get kubeconfig-path --name=${CLUSTER_NAME})

kubectl create -f $MANIFESTS_DIR/kube-flannel.yaml

wait_kind_up

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

#move the pf to the node
mkdir -p /var/run/netns/
export pid="$(docker inspect -f '{{.State.Pid}}' ${CLUSTER_CONTROL_PLANE})"
ln -sf /proc/$pid/ns/net "/var/run/netns/${CLUSTER_NAME}"

sriov_pfs=( /sys/class/net/*/device/sriov_numvfs )

for ifs in "${sriov_pfs[@]}"; do
  ifs_name="${ifs%%/device/*}"
  ifs_name="${ifs_name##*/}"
  ip link set "$ifs_name" netns ${CLUSTER_NAME}
done

# deploy multus
kubectl create -f $MANIFESTS_DIR/multus.yaml

# deploy sriov cni
kubectl create -f $MANIFESTS_DIR/sriov-crd.yaml
kubectl create -f $MANIFESTS_DIR/sriov-cni-daemonset.yaml

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

# ===============
# deploy kubevirt
# ===============
export KUBEVIRT_PROVIDER=external
export DOCKER_PREFIX=${CONTAINER_REGISTRY_HOST}/kubevirt
export DOCKER_TAG=devel
make cluster-build
make cluster-deploy

#removing it since it's crashing with dind because loopback devices are shared with the host
kubectl delete -n kubevirt ds disks-images-provider
sleep 5s #wait a bit so the pods is not being waited for

wait_kubevirt_up

# =========================
# enable sriov feature gate
# =========================
kubectl patch configmap kubevirt-config -n kubevirt --patch "data:
  feature-gates: DataVolumes, CPUManager, LiveMigration, SRIOV"

# delete all virt- pods so that they have a chance to catch up with feature gate change
kubectl get pods -n kubevirt | grep virt | awk '{print $1}' | xargs kubectl delete pods -n kubevirt
wait_kubevirt_up

${CLUSTER_CMD} chmod 666 /dev/vfio/vfio
${CLUSTER_CMD} mount -o remount,rw /sys     # kind remounts it as readonly when it starts, we need it to be writeable

# ========================
# execute functional tests
# ========================

ginko_params="--ginkgo.noColor --junit-output=$ARTIFACTS_PATH/junit.functest.xml --ginkgo.focus=SRIOV --kubeconfig /root/.kube/kind-config-sriov-ci"
FUNC_TEST_ARGS=$ginko_params make functest
