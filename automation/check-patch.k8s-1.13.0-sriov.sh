#!/bin/bash -e
set -x

########################
# This is based on https://github.com/SchSeba/kubevirt-docker

KUBEVIRT_FOLDER=`pwd`

SHARED_DIR="/var/lib/stdci/shared"
SRIOV_JOB_LOCKFILE="${SHARED_DIR}/sriov.lock"
SRIOV_TIMEOUT_SEC="14400" # 4h

function finish {
docker run --rm -e KUBEVIRT_FOLDER=${KUBEVIRT_FOLDER} -v /var/run/docker.sock:/var/run/docker.sock -v `pwd`:/kubevirt -v ${KUBEVIRT_FOLDER}/cluster/k8s-1.13.0-sriov:/root/.kube/ --network host -t sebassch/centos-docker-client clean
}

trap finish EXIT

# serialize all SR-IOV jobs running on the same node
[ -d "${SHARED_DIR}" ] || mkdir -p "${SHARED_DIR}"
touch "$SRIOV_JOB_LOCKFILE"
exec {fd}< "$SRIOV_JOB_LOCKFILE"
flock -e  -w "$SRIOV_TIMEOUT_SEC" "$fd" || {
    echo "ERROR: Timed out after $SRIOV_TIMEOUT_SEC seconds waiting for sriov.lock" >&2
    exit 1
}

losetup -d /dev/loop0 || true

docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -e KUBEVIRT_FOLDER=${KUBEVIRT_FOLDER} -v `pwd`:/kubevirt -v ${KUBEVIRT_FOLDER}/cluster/k8s-1.13.0-sriov:/root/.kube/ --network host -t sebassch/centos-docker-client up

# Wait for nodes to become ready
kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get nodes --no-headers
kubectl_rc=$?
while [ $kubectl_rc -ne 0 ]; do
    echo "Waiting for all nodes to become ready ..."
    kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get nodes --no-headers
    kubectl_rc=$?
    sleep 10
done

# Wait until k8s pods are running
while [ -n "$(kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods --all-namespaces --no-headers | grep -v Running)" ]; do
    echo "Waiting for all pods to enter the Running state ..."
    kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods --all-namespaces --no-headers | >&2 grep -v Running || true
    sleep 10
done

# Make sure all containers are ready
while [ -n "$(kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | grep false)" ]; do
    echo "Waiting for all containers to become ready ..."
    kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers
    sleep 10
done

# move all VF netlink interfaces into kube-master
# ===============================================
devs=($(ls /sys/class/net))
pfs=()
vfs=()

for dev in ${devs[@]}; do
  if [[ -f "/sys/class/net/$dev/device/sriov_numvfs" ]]; then
    pfs+=($dev)
    devfiles=($(ls /sys/class/net/$dev/device/))
    for devfile in ${devfiles[@]}; do
       if [[ $devfile == virtfn* ]]; then
          vfnic=$(ls /sys/class/net/$dev/device/$devfile/net) || true
          if [ ! -z $vfnic ]; then
            vfs+=($vfnic)
          fi
       fi
    done
  fi
done

DOCKER_NAMESPACE=`docker inspect kube-master | grep netns | tr "/" " "  | awk '{print substr($7, 1, length($7)-2)}'`

for ifc in ${vfs[@]}; do
   ip link set $ifc netns ${DOCKER_NAMESPACE}
done

for ifc in ${pfs[@]}; do
   ip link set $ifc netns ${DOCKER_NAMESPACE}
done
# ===============================================

#deploy multus
kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config apply -f cluster/k8s-1.13.0-sriov/manifests/multus.yaml

# configure sriov device plugin
docker exec kube-master ./automation/configure_sriovdp.sh

# deploy sriov services
kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config apply -f cluster/k8s-1.13.0-sriov/manifests/sriov-crd.yaml
kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config apply -f cluster/k8s-1.13.0-sriov/manifests/sriovdp-daemonset.yaml
kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config apply -f cluster/k8s-1.13.0-sriov/manifests/sriov-cni-daemonset.yaml
sleep 10

# Make sure all containers are ready
while [ -n "$(kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | grep false)" ]; do
    echo "Waiting for all containers to become ready ..."
    kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers
    sleep 10
done

# todo: why do we need to ignore it anyway?
docker exec -it kube-master localstore || true

# Build docker containers
docker exec -t kube-master make

docker exec -t kube-master make docker

docker exec -t kube-master make cluster-deploy

# NOTE: this assumes that once at least a single virt- service pops up then
# others will pop up too in quick succession, at least before the first one
# transits to ready state. If it's ever not the case, we may end up exiting
# this function before all virt pods are scheduled and in ready state. If this
# ever happens, we may need to list all services we expect in a kubevirt
# cluster and check each of them is up and running.
function wait_cluster_up {
    # it takes a while for virt-operator to schedule virt pods; wait for at least one of them to pop up
    while [ -z "$(kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods -n kubevirt | grep virt)" ]; do
	sleep 10
    done

    # Make sure all kubevirt containers are ready
    while [ -n "$(kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods -n kubevirt -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | grep false)" ]; do
        echo "Waiting for all containers to become ready ..."
        kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods -n kubevirt -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers
        sleep 10
    done
}

wait_cluster_up

# enable sriov feature gate
kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config patch configmap kubevirt-config -n kubevirt --patch "data:
  feature-gates: DataVolumes, CPUManager, LiveMigration, SRIOV"
kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config get pods -n kubevirt | grep virt | awk '{print $1}' | xargs kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config delete pods -n kubevirt

wait_cluster_up

docker exec -t kube-master ./cluster/k8s-1.13.0-sriov/test.sh
