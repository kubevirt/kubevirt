#!/bin/bash

set -ex

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/manifests"
CI_CONFIGS_DIR="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER"

SRIOV_WORKER_NODES_LABEL=${SRIOV_WORKER_NODES_LABEL:-none}

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

# Using kubectl wait is not realiable if pods restarts
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

_kubectl apply -f $MANIFESTS_DIR/multus.yaml

_kubectl kustomize $MANIFESTS_DIR > $CI_CONFIGS_DIR/sriov-components.yaml
_kubectl apply -f $CI_CONFIGS_DIR/sriov-components.yaml
wait_pods_ready

# Verify that sriov node has sriov VFs allocatable resource

resource_prefix=$(sed -nE 's/.*--resource-prefix=(.*)/\1/p' $CI_CONFIGS_DIR/sriov-components.yaml | head -1)
resource_name=$(sed -nE 's/.*"resourceName":.*"(.*)".*/\1/p' $CI_CONFIGS_DIR/sriov-components.yaml)
numvfs=$(cat $(ls /sys/bus/pci/devices/*/sriov_numvfs | head -1))
for node in $(_kubectl get nodes -l $SRIOV_WORKER_NODES_LABEL -o custom-columns=:.metadata.name --no-headers); do
  wait_allocatable_resource $node "$resource_prefix/$resource_name" $numvfs
done

_kubectl get nodes
_kubectl get pods -A

echo
echo "$KUBEVIRT_PROVIDER cluster is ready"