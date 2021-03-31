#!/bin/bash
set -xe

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/manifests"
CERTCREATOR_PATH="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/certcreator"
KUBECONFIG_PATH="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig"

MASTER_NODE="${CLUSTER_NAME}-control-plane"
WORKER_NODE_ROOT="${CLUSTER_NAME}-worker"
PF_COUNT_PER_NODE=${PF_COUNT_PER_NODE:-1}

OPERATOR_GIT_HASH=8d3c30de8ec5a9a0c9eeb84ea0aa16ba2395cd68  # release-4.4
SRIOV_OPERATOR_NAMESPACE="sriov-network-operator"

[ $PF_COUNT_PER_NODE -le 0 ] && echo "FATAL: PF_COUNT_PER_NODE must be a positive integer" >&2 && exit 1

# This function gets a command and invoke it repeatedly
# until the command return code is zero
function retry {
  local -r tries=$1
  local -r wait_time=$2
  local -r action=$3
  local -r wait_message=$4
  local -r waiting_action=$5

  eval $action
  local return_code=$?
  for i in $(seq $tries); do
    if [[ $return_code -ne 0 ]] ; then
      echo "[$i/$tries] $wait_message"
      eval $waiting_action
      sleep $wait_time
      eval $action
      return_code=$?
    else
      return 0
    fi
  done

  return 1
}

function wait_for_daemonSet {
  local name=$1
  local namespace=$2
  local required_replicas=$3

  if [[ $namespace != "" ]];then
    namespace="-n $namespace"
  fi

  if (( required_replicas < 0 )); then
      echo "DaemonSet $name ready replicas number is not valid: $required_replicas"
      return 1
  fi

  local -r tries=30
  local -r wait_time=10
  wait_message="Waiting for DaemonSet $name to have $required_replicas ready replicas"
  error_message="DaemonSet $name did not have $required_replicas ready replicas"
  action="_kubectl get daemonset $namespace $name -o jsonpath='{.status.numberReady}' | grep -w $required_replicas"

  if ! retry "$tries" "$wait_time" "$action" "$wait_message";then
    echo $error_message
    return 1
  fi

  return  0
}

function wait_k8s_object {
  local -r object_type=$1
  local -r name=$2
  local namespace=$3

  local -r tries=60
  local -r wait_time=3

  local -r wait_message="Waiting for $object_type $name"
  local -r error_message="$object_type $name at $namespace namespace found"

  if [[ $namespace != "" ]];then
    namespace="-n $namespace"
  fi

  local -r action="_kubectl get $object_type $name $namespace -o custom-columns=NAME:.metadata.name --no-headers"

  if ! retry "$tries" "$wait_time" "$action" "$wait_message";then
    echo $error_message
    return  1
  fi

  return 0
}

function _check_all_pods_ready() {
  all_pods_ready_condition=$(_kubectl get pods -A --no-headers -o custom-columns=':.status.conditions[?(@.type == "Ready")].status')
  if [ "$?" -eq 0 ]; then
    pods_not_ready_count=$(grep -cw False <<< "$all_pods_ready_condition")
    if [ "$pods_not_ready_count" -eq 0 ]; then
      return 0
    fi
  fi

  return 1
}

# not using kubectl wait since with the sriov operator the pods get restarted a couple of times and this is
# more reliable
function wait_pods_ready {
  local -r tries=30
  local -r wait_time=10

  local -r wait_message="Waiting for all pods to become ready.."
  local -r error_message="Not all pods were ready after $(($tries*$wait_time)) seconds"

  local -r get_pods='_kubectl get pods --all-namespaces'
  local -r action="_check_all_pods_ready"

  set +x
  trap "set -x" RETURN

  if ! retry "$tries" "$wait_time" "$action" "$wait_message" "$get_pods"; then
    echo $error_message
    return 1
  fi

  echo "all pods are ready"
  return 0
}

function wait_allocatable_resource {
  local -r node=$1
  local resource_name=$2
  local -r expected_value=$3

  local -r tries=48
  local -r wait_time=10

  local -r wait_message="wait for $node node to have allocatable resource: $resource_name: $expected_value"
  local -r error_message="node $node doesnt have allocatable resource $resource_name:$expected_value"

  # it is necessary to add '\' before '.' in the resource name.
  resource_name=$(echo $resource_name | sed s/\\./\\\\\./g)
  local -r action='_kubectl get node $node -ocustom-columns=:.status.allocatable.$resource_name --no-headers | grep -w $expected_value'

  if ! retry $tries $wait_time "$action" "$wait_message"; then
    echo $error_message
    return 1
  fi

  return 0
}

function deploy_multus {
  echo 'Deploying Multus'
  _kubectl create -f $MANIFESTS_DIR/multus.yaml

  echo 'Waiting for Multus deployment to become ready'
  daemonset_name=$(cat $MANIFESTS_DIR/multus.yaml | grep -i daemonset -A 3 | grep -Po '(?<=name:) \S*amd64$')
  daemonset_namespace=$(cat $MANIFESTS_DIR/multus.yaml | grep -i daemonset -A 3 | grep -Po '(?<=namespace:) \S*$' | head -1)
  required_replicas=$(_kubectl get daemonset $daemonset_name -n $daemonset_namespace -o jsonpath='{.status.desiredNumberScheduled}')
  wait_for_daemonSet $daemonset_name $daemonset_namespace $required_replicas

  return 0
}

function deploy_sriov_operator {
  echo 'Downloading the SR-IOV operator'
  operator_path=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/sriov-network-operator-${OPERATOR_GIT_HASH}
  if [ ! -d $operator_path ]; then
    curl -LSs https://github.com/openshift/sriov-network-operator/archive/${OPERATOR_GIT_HASH}/sriov-network-operator.tar.gz | tar xz -C ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/
  fi

  echo 'Installing the SR-IOV operator'
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

  echo 'Generating webhook certificates for the SR-IOV operator webhooks'
  pushd "${CERTCREATOR_PATH}"
    go run . -namespace sriov-network-operator -secret operator-webhook-service -hook operator-webhook -kubeconfig $KUBECONFIG_PATH
    go run . -namespace sriov-network-operator -secret network-resources-injector-secret -hook network-resources-injector -kubeconfig $KUBECONFIG_PATH
  popd

  echo 'Setting caBundle for SR-IOV webhooks'
  wait_k8s_object "validatingwebhookconfiguration" "operator-webhook-config"
  _kubectl patch validatingwebhookconfiguration operator-webhook-config --patch '{"webhooks":[{"name":"operator-webhook.sriovnetwork.openshift.io", "clientConfig": { "caBundle": "'"$(cat $CERTCREATOR_PATH/operator-webhook.cert)"'" }}]}'

  wait_k8s_object "mutatingwebhookconfiguration"   "operator-webhook-config"
  _kubectl patch mutatingwebhookconfiguration operator-webhook-config --patch '{"webhooks":[{"name":"operator-webhook.sriovnetwork.openshift.io", "clientConfig": { "caBundle": "'"$(cat $CERTCREATOR_PATH/operator-webhook.cert)"'" }}]}'

  wait_k8s_object "mutatingwebhookconfiguration"   "network-resources-injector-config"
  _kubectl patch mutatingwebhookconfiguration network-resources-injector-config --patch '{"webhooks":[{"name":"network-resources-injector-config.k8s.io", "clientConfig": { "caBundle": "'"$(cat $CERTCREATOR_PATH/network-resources-injector.cert)"'" }}]}'

  return 0
}

function apply_sriov_node_policy {
  local -r policy_file=$1
  local -r node_pf=$2
  local -r num_vfs=$3

  # Substitute $NODE_PF and $NODE_PF_NUM_VFS and create SriovNetworkNodePolicy CR
  local -r policy=$(NODE_PF=$node_pf NODE_PF_NUM_VFS=$num_vfs envsubst < $policy_file)
  echo "Applying SriovNetworkNodeConfigPolicy:"
  echo "$policy"
  _kubectl create -f - <<< "$policy"

  return 0
}

function move_sriov_pfs_netns_to_node {
  local -r node=$1
  local -r pf_count_per_node=$2
  local -r pid="$(docker inspect -f '{{.State.Pid}}' $node)"
  local pf_array=()

  mkdir -p /var/run/netns/
  ln -sf /proc/$pid/ns/net "/var/run/netns/$node"

  local -r sriov_pfs=( $(find /sys/class/net/*/device/sriov_numvfs) )
  [ "${#sriov_pfs[@]}" -eq 0 ] && echo "FATAL: Could not find available sriov PFs" >&2 && return 1

  for pf in "${sriov_pfs[@]}"; do
    local pf_name="${pf%%/device/*}"
    pf_name="${pf_name##*/}"

    if [ $(echo "${PF_BLACKLIST[@]}" | grep "${pf_name}") ]; then
      continue
    fi

    # In case two clusters started at the same time, they might race on the same PF.
    # The first will manage to assign the PF to its container, and the 2nd will just skip it
    # and try the rest of the PFs available.
    if ip link set "$pf_name" netns "$node"; then
      if timeout 10s bash -c "until ip netns exec $node ip link show $pf_name > /dev/null; do sleep 1; done"; then
        pf_array+=("$pf_name")
        [ "${#pf_array[@]}" -eq "$pf_count_per_node" ] && break
      fi
    fi
  done

  [ "${#pf_array[@]}" -lt "$pf_count_per_node" ] && \
    echo "FATAL: Not enough PFs allocated, PF_BLACKLIST (${PF_BLACKLIST[@]}), PF_COUNT_PER_NODE ${PF_COUNT_PER_NODE}" >&2 && \
    return 1

  echo "${pf_array[@]}"
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

NODE_PFS=($(move_sriov_pfs_netns_to_node "$SRIOV_NODE" "$PF_COUNT_PER_NODE"))

SRIOV_NODE_CMD="docker exec -it -d ${SRIOV_NODE}"
${SRIOV_NODE_CMD} mount -o remount,rw /sys     # kind remounts it as readonly when it starts, we need it to be writeable
${SRIOV_NODE_CMD} chmod 666 /dev/vfio/vfio
_kubectl label node $SRIOV_NODE sriov=true

for pf in "${NODE_PFS[@]}"; do
  docker exec $SRIOV_NODE bash -c "echo 0 > /sys/class/net/$pf/device/sriov_numvfs"
done

deploy_multus
wait_pods_ready

deploy_sriov_operator
wait_pods_ready

# We use just the first suitable pf, for the SriovNetworkNodePolicy manifest.
# We also need the num of vfs because if we don't set this value equals to the total, in case of mellanox
# the sriov operator will trigger a node reboot to update the firmware
NODE_PF=$NODE_PFS
NODE_PF_NUM_VFS=$(docker exec $SRIOV_NODE cat /sys/class/net/$NODE_PF/device/sriov_totalvfs)

POLICY="$MANIFESTS_DIR/network_config_policy.yaml"
apply_sriov_node_policy "$POLICY" "$NODE_PF" "$NODE_PF_NUM_VFS"

# Verify that sriov node has sriov VFs allocatable resource
resource_name=$(sed -n 's/.*resourceName: *//p' $POLICY)
wait_allocatable_resource $SRIOV_NODE "openshift.io/$resource_name" $NODE_PF_NUM_VFS
wait_pods_ready

_kubectl get nodes
_kubectl get pods -n $SRIOV_OPERATOR_NAMESPACE
echo
echo "$KUBEVIRT_PROVIDER cluster is ready"
