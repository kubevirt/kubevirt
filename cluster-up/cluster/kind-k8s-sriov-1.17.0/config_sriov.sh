#!/bin/bash -e
set -x

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/manifests"
CSRCREATORPATH="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/csrcreator"
KUBECONFIG_PATH="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig"

MASTER_NODE="${CLUSTER_NAME}-control-plane"
WORKER_NODE_ROOT="${CLUSTER_NAME}-worker"

OPERATOR_GIT_HASH=8d3c30de8ec5a9a0c9eeb84ea0aa16ba2395cd68  # release-4.4

function ensure_cr {
  crd=$1
  cr=$2
  namespace=$3

  intervals=30
  timeout=3

  if [[ $namespace != "" ]];then
    namespace="-n $namespace"
  fi

  count=0
  until _kubectl get $crd $cr $namespace; do
      ((count++)) && ((count == intervals)) && echo "$crd CR not found" && exit 1
      echo "[$count/$intervals] Waiting for $crd CR.."
      sleep $timeout
  done
}

function ensure_caBundle {
  webhook_configuration_type=$1
  webhook_configuration=$2
  hash=$3

  intervals=6
  timeout=10

  count=0
  current_cab=$(_kubectl get $webhook_configuration_type $webhook_configuration -ocustom-columns=CAB:.webhooks[*].clientConfig.caBundle --no-headers)
  target_cab=$hash
  until [[ $current_cab == $target_cab ]] ; do
      ((count++)) && ((count == intervals)) && echo "$webhook_configuration_type caBundle did not changed" && exit 1
      echo "[$count/$intervals] Waiting for $webhook_configuration_type-$webhook_configuration caBundle update.."
      sleep $timeout
  done
}
function wait_for_taint {
  taint=$1

  intervals=30
  timeout=10

  count=0
  until [[ $(_kubectl  get nodes -ocustom-columns=taints:.spec.taints[*].effect --no-headers | grep -i $taint) != "" ]]; do
    ((count++)) && ((count == intervals)) && echo "Taint $taint did not removed in after $intervals tries" && exit 1
    echo "[$count/$intervals] Waiting for taint $taint absence"
    _kubectl get nodes -ocustom-columns=NAME:.metadata.name,TAINTS:.spec.taints[*].effect --no-headers
    sleep $timeout
  done
}

function wait_for_taint_absence {
  taint=$1

  intervals=30
  timeout=10

  count=0
  until [[ $(_kubectl  get nodes -ocustom-columns=taints:.spec.taints[*].effect --no-headers | grep -i $taint) == "" ]]; do
    ((count++)) && ((count == intervals)) && echo "Taint $taint did not removed in after $intervals tries" && exit 1
    echo "[$count/$intervals] Waiting for taint $taint absence"
    _kubectl get nodes -ocustom-columns=NAME:.metadata.name,TAINTS:.spec.taints[*].effect --no-headers
    sleep $timeout
  done
}

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
    make deploy-setup-k8s SHELL=/bin/bash  # on prow nodes the default shell is dash and some commands are not working
  popd

  pushd "${CSRCREATORPATH}"
    go run . -namespace sriov-network-operator -secret operator-webhook-service -hook operator-webhook -kubeconfig $KUBECONFIG_PATH
    go run . -namespace sriov-network-operator -secret network-resources-injector-secret -hook network-resources-injector -kubeconfig $KUBECONFIG_PATH
  popd
}

# The first worker needs to be handled specially as it has no ending number, and sort will not work
# We add the 0 to it and we remove it if it's the candidate worker
WORKER=$(_kubectl get nodes | grep $WORKER_NODE_ROOT | sed "s/\b$WORKER_NODE_ROOT\b/${WORKER_NODE_ROOT}0/g" | sort -r | awk 'NR==1 {print $1}')
if [[ -z "$WORKER" ]]; then
  SRIOV_NODE=$MASTER_NODE
else
  SRIOV_NODE=$WORKER
fi

# this is to remove the ending 0 in case the candidate worker is the first one
if [[ "$SRIOV_NODE" == "${WORKER_NODE_ROOT}0" ]]; then
  SRIOV_NODE=${WORKER_NODE_ROOT}
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

_kubectl label node $SRIOV_NODE sriov=true

wait_pods_ready

# Ensure webook-configuration object created
ensure_cr "validatingwebhookconfiguration" "operator-webhook-config"
ensure_cr "mutatingwebhookconfiguration"   "operator-webhook-config"
ensure_cr "mutatingwebhookconfiguration"   "network-resources-injector-config"

_kubectl patch validatingwebhookconfiguration operator-webhook-config --patch '{"webhooks":[{"name":"operator-webhook.sriovnetwork.openshift.io", "clientConfig": { "caBundle": "'"$(cat $CSRCREATORPATH/operator-webhook.cert)"'" }}]}'
_kubectl patch mutatingwebhookconfiguration network-resources-injector-config --patch '{"webhooks":[{"name":"network-resources-injector-config.k8s.io", "clientConfig": { "caBundle": "'"$(cat $CSRCREATORPATH/network-resources-injector.cert)"'" }}]}'
_kubectl patch mutatingwebhookconfiguration operator-webhook-config --patch '{"webhooks":[{"name":"operator-webhook.sriovnetwork.openshift.io", "clientConfig": { "caBundle": "'"$(cat $CSRCREATORPATH/operator-webhook.cert)"'" }}]}'

# Ensure caBundle is configured with new certificates
ensure_caBundle "validatingwebhookconfiguration" "operator-webhook-config"           "$(cat $CSRCREATORPATH/operator-webhook.cert)"
ensure_caBundle "mutatingwebhookconfiguration"   "operator-webhook-config"           "$(cat $CSRCREATORPATH/operator-webhook.cert)"
ensure_caBundle "mutatingwebhookconfiguration"   "network-resources-injector-config" "$(cat $CSRCREATORPATH/network-resources-injector.cert)"

# Wait caBundle reconcile to finish by waiting for the "NoSchedule" taint
# to present and then absent.
wait_for_taint "NoSchedule"
wait_for_taint_absence "NoSchedule"

# Substitute NODE_PF and NODE_PF_NUM_VFS then create SriovNetworkNodePolicy CR
envsubst < $MANIFESTS_DIR/network_config_policy.yaml | _kubectl create -f -

sriov_operator_namespace="sriov-network-operator"

# Ensure SriovNetworkNodePolicy CR is created
policy_name=$(cat $MANIFESTS_DIR/network_config_policy.yaml | grep 'name:' | awk '{print $2}')
ensure_cr "SriovNetworkNodePolicy" $policy_name $sriov_operator_namespace

# Wait for cni and device-plugin pods to be ready
_kubectl wait pods -n $sriov_operator_namespace -l app=sriov-cni           --for condition=Ready --timeout 300s
_kubectl wait pods -n $sriov_operator_namespace -l app=sriov-device-plugin --for condition=Ready --timeout 300s

# Wait for nodes NoSchedule taint to be removed
wait_for_taint_absence "NoSchedule"

${SRIOV_NODE_CMD} chmod 666 /dev/vfio/vfio
