#!/bin/bash -e
set -x

CONTROL_PLANE_CMD="docker exec -it -d ${CLUSTER_NAME}-control-plane"
MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/manifests"
OPERATOR_GIT_HASH=b3ab84a316e16df392fbe9e07dbe0667ad075855

# not using kubectl wait since with the sriov operator the pods get restarted a couple of times and this is
# more reliable
function wait_pods_ready {
    while [ -n "$(kubectl get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | grep false)" ]; do
        echo "Waiting for all pods to become ready ..."
        kubectl get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers
        sleep 10
    done
}

function deploy_sriov_operator {
  OPERATOR_PATH=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/sriov-network-operator-${OPERATOR_GIT_HASH}
  if [[ ! -d $OPERATOR_PATH ]]; then
    curl -L https://github.com/openshift/sriov-network-operator/archive/${OPERATOR_GIT_HASH}/sriov-network-operator.tar.gz | tar xz -C ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/
  fi

  pushd $OPERATOR_PATH
    # TODO: right now in CI we need to use upstream sriov cni in order to have this
    # https://github.com/intel/sriov-cni/pull/88 available. This can be removed once the feature will
    # be merged in openshift sriov operator. We need latest since that feature was not tagged yet
    sed -i '/SRIOV_CNI_IMAGE/!b;n;c\              value: nfvpe\/sriov-cni' ./deploy/operator.yaml

    # on prow nodes the default shell is dash and some commands are not working
    make deploy-setup-k8s SHELL=/bin/bash OPERATOR_EXEC=kubectl
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
wait_pods_ready

${CONTROL_PLANE_CMD} mount -o remount,rw /sys     # kind remounts it as readonly when it starts, we need it to be writeable

deploy_sriov_operator

kubectl label node sriov-control-plane node-role.kubernetes.io/worker=
kubectl label node sriov-control-plane sriov=true 
envsubst < $MANIFESTS_DIR/network_config_policy.yaml | kubectl create -f -


wait_pods_ready

${CONTROL_PLANE_CMD} chmod 666 /dev/vfio/vfio
