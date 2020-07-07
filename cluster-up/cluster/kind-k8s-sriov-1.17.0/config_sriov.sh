#!/bin/bash -e
set -x

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/manifests"
CSRCREATORPATH="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/csrcreator"
KUBECONFIG_PATH="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig"

MASTER_NODE="${CLUSTER_NAME}-control-plane"
WORKER_NODE_ROOT="${CLUSTER_NAME}-worker"

OPERATOR_GIT_HASH=8d3c30de8ec5a9a0c9eeb84ea0aa16ba2395cd68  # release-4.4

# This function gets a command string and invoke it
# until the command returns an empty string or until timeout
function retry {
  local -r tries=$1
  local -r wait_time=$2
  local -r action=$3
  local -r wait_message=$4

  local result=$(eval $action)
  for i in $(seq $tries); do
    if [[ -z $result ]] ; then
      echo "[$i/$tries] $wait_message"
      sleep $wait_time
      result=$(eval $action)
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
      echo "DaemonSet $name ready replicas number is not valid: $required_replicas" && return 1
  fi

  local -r tries=30
  local -r wait_time=10
  wait_message="Waiting for DaemonSet $name to have $required_replicas ready replicas"
  error_message="DaemonSet $name did not have $required_replicas ready replicas"
  action="_kubectl get daemonset $namespace $name -o jsonpath='{.status.numberReady}' | grep -w $required_replicas"

  retry "$tries" "$wait_time" "$action" "$wait_message" && return  0
  echo $error_message && return 1
}

function wait_pod {
  local namespace=$1
  local label=$2

  local -r tries=60
  local -r wait_time=5

  local -r wait_message="Waiting for pods with $label to create"
  local -r error_message="Pods with  label $label at $namespace namespace found"

  if [[ $namespace != "" ]];then
    namespace="-n $namespace"
  fi

  if [[ $label != "" ]];then
    label="-l $label"
  fi

  local -r action="_kubectl get pod $namespace $label -o custom-columns=NAME:.metadata.name --no-headers"

  retry "$tries" "$wait_time" "$action" "$wait_message" && return  0
  echo $error_message && return 1
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

  retry "$tries" "$wait_time" "$action" "$wait_message" && return 0
  echo $error_message && return  1
}


function is_taint_absence {
  local -r taint=$1

  result=$(_kubectl get nodes -o jsonpath="{.items[*].spec.taints[?(@.effect == \"$taint\")].effect}" || echo error)
  if [[ -z $result ]]; then
    echo "not-present"
  fi
}

function wait_for_taint_absence {
  local -r taint=$1

  local -r tries=60
  local -r wait_time=5

  local -r wait_message="Waiting for $taint taint absence"
  local -r error_message="Taint $taint $name did not removed"
  local -r action="is_taint_absence $taint"

  retry "$tries" "$wait_time" "$action" "$wait_message" && return 0
  echo $error_message && return 1
}

function wait_for_taint {
  local -r taint=$1

  local -r tries=60
  local -r wait_time=5

  local -r wait_message="Waiting for $taint taint to present"
  local -r error_message="Taint $taint $name did not present"
  local -r action="_kubectl get nodes -o custom-columns=taints:.spec.taints[*].effect --no-headers | grep -i $taint"

  retry "$tries" "$wait_time" "$action" "$wait_message" && return 0
  echo $error_message && return 1
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

function wait_allocatable_resource {
  local -r node=$1
  local resource_name=$2
  local -r expected_value=$3

  local -r tries=30
  local -r wait_time=10

  local -r wait_message="wait for $node node to have allocatable resource: $resource_name: $expected_value"
  local -r error_message="node $node doesnt have allocatable resource $resource_name:$expected_value"

  # In order to project spesific resource name with -o custom-columns
  # it is necessary to add '\' before '.' in the resource name.
  resource_name=$(echo $resource_name | sed s/\\./\\\\\./g)
  local -r action='_kubectl get node $node -ocustom-columns=:.status.allocatable.$resource_name --no-headers | grep -w $expected_value'

  retry $tries $wait_time "$action" "$wait_message" && return 0
  echo $error_message && return 1
}

function deploy_multus {
  echo 'Deploying Multus'
  _kubectl create -f $MANIFESTS_DIR/multus.yaml

  echo 'Waiting for Multus deployment to become ready'
  daemonset_name=$(cat $MANIFESTS_DIR/multus.yaml | grep -i daemonset -A 3 | grep -Po '(?<=name:) \S*amd64$')
  daemonset_namespace=$(cat $MANIFESTS_DIR/multus.yaml | grep -i daemonset -A 3 | grep -Po '(?<=namespace:) \S*$' | head -1)
  wait_k8s_object "daemonset" $daemonset_name $daemonset_namespace || return 1

  required_replicas=$(_kubectl get daemonset kube-multus-ds-amd64 -n  kube-system -o jsonpath='{.status.desiredNumberScheduled}')
  wait_for_daemonSet $daemonset_name $daemonset_namespace $required_replicas || return 1

  return 0
}

function deploy_sriov_operator {
  echo 'Downloading the SR-IOV operator'
  operator_path=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/sriov-network-operator-${OPERATOR_GIT_HASH}
  if [ ! -d $operator_path ]; then
    curl -L https://github.com/openshift/sriov-network-operator/archive/${OPERATOR_GIT_HASH}/sriov-network-operator.tar.gz | tar xz -C ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/
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
  pushd "${CSRCREATORPATH}"
    go run . -namespace sriov-network-operator -secret operator-webhook-service -hook operator-webhook -kubeconfig $KUBECONFIG_PATH || return 1
    go run . -namespace sriov-network-operator -secret network-resources-injector-secret -hook network-resources-injector -kubeconfig $KUBECONFIG_PATH || return 1
  popd

  echo 'Setting caBundle for SR-IOV webhooks'
  wait_k8s_object "validatingwebhookconfiguration" "operator-webhook-config" || return 1
  _kubectl patch validatingwebhookconfiguration operator-webhook-config --patch '{"webhooks":[{"name":"operator-webhook.sriovnetwork.openshift.io", "clientConfig": { "caBundle": "'"$(cat $CSRCREATORPATH/operator-webhook.cert)"'" }}]}'

  wait_k8s_object "mutatingwebhookconfiguration"   "operator-webhook-config" || return 1
  _kubectl patch mutatingwebhookconfiguration operator-webhook-config --patch '{"webhooks":[{"name":"operator-webhook.sriovnetwork.openshift.io", "clientConfig": { "caBundle": "'"$(cat $CSRCREATORPATH/operator-webhook.cert)"'" }}]}'

  wait_k8s_object "mutatingwebhookconfiguration"   "network-resources-injector-config" || return 1
  _kubectl patch mutatingwebhookconfiguration network-resources-injector-config --patch '{"webhooks":[{"name":"network-resources-injector-config.k8s.io", "clientConfig": { "caBundle": "'"$(cat $CSRCREATORPATH/network-resources-injector.cert)"'" }}]}'

  # Since sriov-operator doesnt have a condition or Status to indicate if
  # 'operator-webhook' and 'network-resources-injector' webhooks certificates are
  # configured, in order to check if caBundle reconcile is finished it is necessary
  # to wait for the "NoSchedule" taint to present and then absent.
  taint="NoSchedule"
  wait_for_taint "$taint" || echo "Taint $taint did not present on nodes after setting caBundle for sriov webhooks"
  wait_for_taint_absence "$taint" || return 1

  return 0
}

function apply_sriov_node_policy {
  policy_file=$1

  SRIOV_OPERATOR_NAMESPACE="sriov-network-operator"
  SRIOV_DEVICE_PLUGIN_LABEL="app=sriov-device-plugin"
  SRIOV_CNI_LABEL="app=sriov-cni"

  echo "Applying SriovNetworkNodeConfigPolicy:"
  cat $policy_file
  # Substitute $NODE_PF and $NODE_PF_NUM_VFS and create SriovNetworkNodePolicy CR
  envsubst < $policy_file | _kubectl create -f -

  # Ensure SriovNetworkNodePolicy CR is created
  policy_name=$(cat $policy_file | grep -Po '(?<=name:) \S*')
  wait_k8s_object "SriovNetworkNodePolicy" "$policy_name" "$SRIOV_OPERATOR_NAMESPACE"  || return 1

  # Wait for sriov-operator to reconcile SriovNodeNetworkPolicy
  # and create cni and device-plugin pods
  wait_pod $SRIOV_OPERATOR_NAMESPACE $SRIOV_CNI_LABEL  || return 1
  wait_pod $SRIOV_OPERATOR_NAMESPACE $SRIOV_DEVICE_PLUGIN_LABEL || return 1

  # Wait for cni and device-plugin pods to be ready
  _kubectl wait pods -n $SRIOV_OPERATOR_NAMESPACE -l $SRIOV_CNI_LABEL           --for condition=Ready --timeout 10m
  _kubectl wait pods -n $SRIOV_OPERATOR_NAMESPACE -l $SRIOV_DEVICE_PLUGIN_LABEL --for condition=Ready --timeout 10m

  # Since SriovNodeNetworkPolicy doesnt have Status to indicate if its
  # configured successfully, it is necessary to wait for the "NoSchedule"
  # taint to present and then absent.
  taint="NoSchedule"
  wait_for_taint "$taint" || echo "Taint $taint did not present on nodes after creating SriovNodeNetworkPolicy"
  wait_for_taint_absence "$taint" || return  1

  return 0
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

SRIOV_NODE_CMD="docker exec -it -d ${SRIOV_NODE}"
${SRIOV_NODE_CMD} mount -o remount,rw /sys     # kind remounts it as readonly when it starts, we need it to be writeable
${SRIOV_NODE_CMD} chmod 666 /dev/vfio/vfio
_kubectl label node $SRIOV_NODE sriov=true

deploy_multus || exit 1
wait_pods_ready

deploy_sriov_operator || exit 1
wait_pods_ready

# Substitute NODE_PF and NODE_PF_NUM_VFS then create SriovNetworkNodePolicy CR
policy="$MANIFESTS_DIR/network_config_policy.yaml"
apply_sriov_node_policy "$policy" || exit 1
wait_pods_ready

# Verify that sriov node has sriov VFs allocatable resource
resource_name=$(cat $policy | grep 'resourceName:' | awk '{print $2}')
wait_allocatable_resource $SRIOV_NODE "openshift.io/$resource_name" $NODE_PF_NUM_VFS || exit 1

_kubectl get nodes
_kubectl get pods -n $SRIOV_OPERATOR_NAMESPACE
echo
echo "$KUBEVIRT_PROVIDER cluster is ready"
