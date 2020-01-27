#!/bin/bash -e
set -x

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/manifests"

MASTER_NODE="${CLUSTER_NAME}-control-plane"
FIRST_WORKER_NODE="${CLUSTER_NAME}-worker"

OPERATOR_GIT_HASH=b3ab84a316e16df392fbe9e07dbe0667ad075855
INJECTOR_GIT_HASH=9ffd768cb7886072e81df3ac78ba2997810ceb55

# not using kubectl wait since with the sriov operator the pods get restarted a couple of times and this is
# more reliable
function wait_pods_ready {
    while [ -n "$(_kubectl get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | grep false)" ]; do
        echo "Waiting for all pods to become ready ..."
        _kubectl get pods --all-namespaces -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers
        sleep 10
    done
}

function deploy_sriov_operator {
  operator_path=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/sriov-network-operator-${OPERATOR_GIT_HASH}
  if [ ! -d $operator_path ]; then
    curl -L https://github.com/openshift/sriov-network-operator/archive/${OPERATOR_GIT_HASH}/sriov-network-operator.tar.gz | tar xz -C ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/
  fi

  pushd $operator_path
    # TODO: right now in CI we need to use upstream sriov cni in order to have this
    # https://github.com/intel/sriov-cni/pull/88 available. This can be removed once the feature will
    # be merged in openshift sriov operator. We need latest since that feature was not tagged yet
    sed -i '/SRIOV_CNI_IMAGE/!b;n;c\              value: nfvpe\/sriov-cni' ./deploy/operator.yaml
    sed -i 's#image: quay.io/openshift/origin-sriov-network-operator$#image: quay.io/openshift/origin-sriov-network-operator:4.2#' ./deploy/operator.yaml
    sed -i 's#value: quay.io/openshift/origin-sriov-network-config-daemon$#value: quay.io/openshift/origin-sriov-network-config-daemon:4.2#' ./deploy/operator.yaml
    # on prow nodes the default shell is dash and some commands are not working
    make deploy-setup-k8s SHELL=/bin/bash OPERATOR_EXEC="${KUBECTL}"
  popd
}

function deploy_network_resource_injector {
  webhook_path=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/network-resources-injector-${INJECTOR_GIT_HASH}
  if [ ! -d $webhook_path ]; then
    curl -L https://github.com/intel/network-resources-injector/archive/${INJECTOR_GIT_HASH}/network-resources-injector.tar.gz | tar xz -C ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/
  fi

  pushd $webhook_path
    make image
    docker tag network-resources-injector localhost:5000/network-resources-injector
    sed -i 's#image: network-resources-injector:latest#image: registry:5000/network-resources-injector:latest#' ./deployments/server.yaml
    docker push localhost:5000/network-resources-injector
    _kubectl apply -f ./deployments/auth.yaml
    _kubectl apply -f ./deployments/server.yaml
  popd
}


if [[ -z "$(_kubectl get nodes | grep $FIRST_WORKER_NODE)" ]]; then
  SRIOV_NODE=$MASTER_NODE
else
  SRIOV_NODE=$FIRST_WORKER_NODE
fi

#move the pf to the node
mkdir -p /var/run/netns/
export pid="$(docker inspect -f '{{.State.Pid}}' $SRIOV_NODE)"
ln -sf /proc/$pid/ns/net "/var/run/netns/$SRIOV_NODE"

sriov_pfs=( /sys/class/net/*/device/sriov_numvfs )


for ifs in "${sriov_pfs[@]}"; do
  ifs_name="${ifs%%/device/*}"
  ifs_name="${ifs_name##*/}"

  if [ $(echo "${PF_BLACKLIST[@]}" | grep -q "${ifs_name}") ]; then
    continue
  fi

  # We set the variable below only in the first iteration as we need only one PF
  # to inject into the Network Configuration manifest. We need to move all pfs to
  # the node's namespace and for that reason we do not interrupt the loop.
  if [ -z "$NODE_PF" ]; then
    # These values are used to populate the network definition policy yaml.
    # We just use the first suitable pf
    # We need the num of vfs because if we don't set this value equals to the total, in case of mellanox
    # the sriov operator will trigger a node reboot to update the firmware
    export NODE_PF="$ifs_name"
    export NODE_PF_NUM_VFS=$(cat /sys/class/net/"$NODE_PF"/device/sriov_totalvfs)
  fi
  ip link set "$ifs_name" netns "$SRIOV_NODE"
done


# deploy multus
_kubectl create -f $MANIFESTS_DIR/multus.yaml

# give them some time to create pods before checking pod status
sleep 10

# make sure all containers are ready
wait_pods_ready

SRIOV_NODE_CMD="docker exec -it -d ${SRIOV_NODE}"

${SRIOV_NODE_CMD} mount -o remount,rw /sys     # kind remounts it as readonly when it starts, we need it to be writeable

deploy_sriov_operator

_kubectl label node $SRIOV_NODE node-role.kubernetes.io/worker=
_kubectl label node $SRIOV_NODE sriov=true
envsubst < $MANIFESTS_DIR/network_config_policy.yaml | _kubectl create -f -


wait_pods_ready

deploy_network_resource_injector

# give the injector installer some time to create pods before checking pod status
sleep 5
wait_pods_ready

${SRIOV_NODE_CMD} chmod 666 /dev/vfio/vfio