#!/bin/bash -e
set -x

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/manifests"
CSRCREATORPATH="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/csrcreator"

MASTER_NODE="${CLUSTER_NAME}-control-plane"
FIRST_WORKER_NODE="${CLUSTER_NAME}-worker"

OPERATOR_GIT_HASH=8d3c30de8ec5a9a0c9eeb84ea0aa16ba2395cd68  # release-4.4

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
    export RELEASE_VERSION=4.4
    export SRIOV_NETWORK_OPERATOR_IMAGE=quay.io/openshift/origin-sriov-network-operator:${RELEASE_VERSION}
    export SRIOV_NETWORK_CONFIG_DAEMON_IMAGE=quay.io/openshift/origin-sriov-network-config-daemon:${RELEASE_VERSION}
    export SRIOV_NETWORK_WEBHOOK_IMAGE=quay.io/openshift/origin-sriov-network-webhook:${RELEASE_VERSION}
    export NETWORK_RESOURCES_INJECTOR_IMAGE=quay.io/openshift/origin-sriov-dp-admission-controller:${RELEASE_VERSION}
    export SRIOV_CNI_IMAGE=quay.io/openshift/origin-sriov-cni:${RELEASE_VERSION}
    export SRIOV_DEVICE_PLUGIN_IMAGE=quay.io/openshift/origin-sriov-network-device-plugin:${RELEASE_VERSION}
    export OPERATOR_EXEC=${KUBECTL}
    export SHELL=/bin/bash  # on prow nodes the default shell is dash and some commands are not working
    make deploy-setup-k8s
  popd

  pushd "${CSRCREATORPATH}" 
    go run . -namespace sriov-network-operator -secret operator-webhook-service -hook operator-webhook
    go run . -namespace sriov-network-operator -secret network-resources-injector-secret -hook network-resources-injector
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
_kubectl apply -f https://raw.githubusercontent.com/intel/multus-cni/master/images/multus-daemonset.yml

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

# we need to sleep as the configurations below need to appear
sleep 30

_kubectl patch validatingwebhookconfiguration operator-webhook-config --patch '{"webhooks":[{"name":"operator-webhook.sriovnetwork.openshift.io", "clientConfig": { "caBundle": "'"$(cat $CSRCREATORPATH/operator-webhook.cert)"'" }}]}'
_kubectl patch mutatingwebhookconfiguration network-resources-injector-config --patch '{"webhooks":[{"name":"network-resources-injector-config.k8s.io", "clientConfig": { "caBundle": "'"$(cat $CSRCREATORPATH/network-resources-injector.cert)"'" }}]}'
_kubectl patch mutatingwebhookconfiguration operator-webhook-config --patch '{"webhooks":[{"name":"operator-webhook.sriovnetwork.openshift.io", "clientConfig": { "caBundle": "'"$(cat $CSRCREATORPATH/operator-webhook.cert)"'" }}]}'


${SRIOV_NODE_CMD} chmod 666 /dev/vfio/vfio

