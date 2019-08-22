#!/bin/bash -e
set -x

CONTROL_PLANE_CMD="docker exec -it -d ${CLUSTER_NAME}-control-plane"
MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/manifests"

function wait_containers_ready {
    echo "Waiting for all containers to become ready ..."
    kubectl wait --for=condition=Ready pod --all -n kube-system --timeout 12m
}

function deploy_sriov_operator {
  OPERATOR_PATH=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/sriov-network-operator
  if [[ ! -d $OPERATOR_PATH ]]; then
    git clone https://github.com/openshift/sriov-network-operator.git $OPERATOR_PATH
  fi

  pushd $OPERATOR_PATH
    export OPERATOR_EXEC=kubectl
    # on prow nodes the default shell is dash and some commands are not working
    make deploy-setup-k8s SHELL=/bin/bash
  popd
}

#move the pf to the node
mkdir -p /var/run/netns/
export pid="$(docker inspect -f '{{.State.Pid}}' ${CLUSTER_NAME}-control-plane)"
ln -sf /proc/$pid/ns/net "/var/run/netns/${CLUSTER_NAME}-control-plane"

sriov_pfs=( /sys/class/net/*/device/sriov_numvfs )

counter=0
for ifs in "${sriov_pfs[@]}"; do
  ifs_name="${ifs%%/device/*}"
  ifs_name="${ifs_name##*/}"

  if  [[ "$counter" -eq 0 ]]; then
    # These values are used to populate the network definition policy yaml. 
    # We need the num of vfs because if we don't set this value equals to the total, in case of mellanox 
    # the sriov operator will trigger a node reboot to update the firmware
    export FIRST_PF="$ifs_name"
    export FIRST_PF_NUM_VFS=$(cat /sys/class/net/"$FIRST_PF"/device/sriov_totalvfs)
  fi
  ip link set "$ifs_name" netns "${CLUSTER_NAME}-control-plane"
  counter=$((counter+1))
done

# deploy multus
kubectl create -f $MANIFESTS_DIR/multus.yaml

# give them some time to create pods before checking pod status
sleep 10

# make sure all containers are ready
wait_containers_ready

${CONTROL_PLANE_CMD} mount -o remount,rw /sys     # kind remounts it as readonly when it starts, we need it to be writeable

deploy_sriov_operator

kubectl label node sriov-control-plane node-role.kubernetes.io/worker=
kubectl label node sriov-control-plane sriov=true 
kubectl wait --for=condition=Ready pod --all -n sriov-network-operator --timeout 6m

envsubst < $MANIFESTS_DIR/network_policy.yaml | kubectl create -f -

sleep 5 #let the daemons appear
SRIOVCNI_DAEMON_POD=$(kubectl get pods -n sriov-network-operator | grep sriov-cni | awk '{print $1}')
kubectl wait --for=condition=Ready -n sriov-network-operator pod $SRIOVCNI_DAEMON_POD --timeout 3m

SRIOVDEVICEPL_DAEMON_POD=$(kubectl get pods -n sriov-network-operator | grep sriov-device | awk '{print $1}')
kubectl wait --for=condition=Ready -n sriov-network-operator pod $SRIOVDEVICEPL_DAEMON_POD --timeout 3m

${CONTROL_PLANE_CMD} chmod 666 /dev/vfio/vfio

