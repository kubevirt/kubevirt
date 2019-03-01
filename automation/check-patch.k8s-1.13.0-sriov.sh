#!/bin/bash -e
set -x

#############################################################
# This is based on https://github.com/SchSeba/kubevirt-docker
#############################################################

KUBEVIRT_FOLDER=`pwd`
MANIFESTS_DIR="cluster/k8s-1.13.0-sriov/manifests"
ARTIFACTS_DIR="$KUBEVIRT_FOLDER/exported-artifacts"

SHARED_DIR="/var/lib/stdci/shared"
SRIOV_JOB_LOCKFILE="${SHARED_DIR}/sriov.lock"
SRIOV_TIMEOUT_SEC="14400" # 4h

runcmd="docker run --rm -e KUBEVIRT_FOLDER=${KUBEVIRT_FOLDER} -v /var/run/docker.sock:/var/run/docker.sock -v ${KUBEVIRT_FOLDER}:/kubevirt -v ${KUBEVIRT_FOLDER}/cluster/k8s-1.13.0-sriov:/root/.kube/ -v /lib/modules:/lib/modules --network host -t sebassch/centos-docker-client"
mastercmd="docker exec kube-master"
kubectl="kubectl --kubeconfig cluster/k8s-1.13.0-sriov/config"

function wait_containers_ready {
    # wait until all containers are ready
    while [ -n "$($kubectl get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | grep false)" ]; do
        echo "Waiting for all containers to become ready ..."
        $kubectl get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers
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
    while [ -z "$($kubectl get pods -n kubevirt | grep virt)" ]; do
        echo "Waiting for all pods to create ..."
        $kubectl get pods -n kubevirt | grep virt
	sleep 10
    done

    wait_containers_ready
}

function collect_artifacts {
    mkdir -p "$ARTIFACTS_DIR"
    $mastercmd journalctl -xe > "$ARTIFACTS_DIR/journalctl-xe.log"
}

function finish {
    collect_artifacts
    $runcmd clean
}

trap finish EXIT

# serialize all sriov jobs running on the same ci node
[ -d "${SHARED_DIR}" ] || mkdir -p "${SHARED_DIR}"
touch "$SRIOV_JOB_LOCKFILE"
exec {fd}< "$SRIOV_JOB_LOCKFILE"
flock -e  -w "$SRIOV_TIMEOUT_SEC" "$fd" || {
    echo "ERROR: Timed out after $SRIOV_TIMEOUT_SEC seconds waiting for sriov.lock" >&2
    exit 1
}

# ================
# bring up cluster
# ================
$runcmd up

# wait for nodes to become ready
until $kubectl get nodes --no-headers
do
    echo "Waiting for all nodes to become ready ..."
    sleep 10
done

# wait until k8s pods are running
while [ -n "$($kubectl get pods --all-namespaces --no-headers | grep -v Running)" ]; do
    echo "Waiting for all pods to enter the Running state ..."
    $kubectl get pods --all-namespaces --no-headers | >&2 grep -v Running || true
    sleep 10
done

# wait until all containers are ready
wait_containers_ready

# ===============================================
# move all VF netlink interfaces into kube-master
# ===============================================
DOCKER_NAMESPACE=`docker inspect kube-master | grep netns | tr "/" " "  | awk '{print substr($7, 1, length($7)-2)}'`

# Instead of dealing with `setns` from within a chroot, we spawn a privileged
# container with host network. Since docker socket is mounted from the host,
# the container is actually being created on the host itself and have access to
# the different namespaces.
# We set MAC addresses for all VFs because some NICs leave their VFs with
# all-zeroes addresses. We use a common MAC prefix from Virtualbox for all  of
# them. And we assume that the number of VFs per node is not higher than 255.
docker run -i --privileged --net=host --rm \
    -v /run/docker/netns/:/var/run/netns/ centos:7 /bin/bash <<EOF
    set -x
    yum install -y iproute
    sriov_vfs=( /sys/class/net/*/device/virtfn* )
    i=0
    for vf in "\${sriov_vfs[@]}"; do
      ifs_arr=( "\$vf"/net/* )
      for ifs in "\${ifs_arr[@]}"; do
          ifs_name="\${ifs%%\/net\/*}"
          ifs_name="\${ifs##*\/}"
          ip link set dev "\$ifs_name" down
          ip link set dev "\$ifs_name" address 0a:00:27:00:00:\$(printf "%x\\n" "\$i")
          ip link set dev "\$ifs_name" up
          ip link set "\$ifs_name" netns "$DOCKER_NAMESPACE"
          i=\$((\$i+1))
      done
    done

    sriov_pfs=( /sys/class/net/*/device/sriov_numvfs )
    for ifs in "\${sriov_pfs[@]}"; do
      ifs_name="\${ifs%%/device/*}"
      ifs_name="\${ifs_name##*/}"
      ip link set "\$ifs_name" netns "$DOCKER_NAMESPACE"
    done
EOF

# ========================
# deploy SR-IOV components
# ========================

# deploy multus
$kubectl apply -f $MANIFESTS_DIR/multus.yaml

# deploy sriov cni
$kubectl apply -f $MANIFESTS_DIR/sriov-crd.yaml
$kubectl apply -f $MANIFESTS_DIR/sriov-cni-daemonset.yaml

# deploy sriov device plugin
$mastercmd ./automation/configure_sriovdp.sh
$kubectl apply -f $MANIFESTS_DIR/sriovdp-daemonset.yaml

# give them some time to create pods before checking pod status
sleep 10

# make sure all containers are ready
wait_containers_ready

# set up local volume directories
losetup -d /dev/loop0 || true
$mastercmd localstore || true  # it will fail trying to mknode /dev/loop0

# ===============
# deploy kubevirt
# ===============
$mastercmd make
$mastercmd make docker
$mastercmd make cluster-deploy
wait_kubevirt_up

# =========================
# enable sriov feature gate
# =========================
$kubectl patch configmap kubevirt-config -n kubevirt --patch "data:
  feature-gates: DataVolumes, CPUManager, LiveMigration, SRIOV"

# delete all virt- pods so that they have a chance to catch up with feature gate change
$kubectl get pods -n kubevirt | grep virt | awk '{print $1}' | xargs $kubectl delete pods -n kubevirt
wait_kubevirt_up

# ========================
# execute functional tests
# ========================
$mastercmd ./cluster/k8s-1.13.0-sriov/test.sh
