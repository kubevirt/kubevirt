#!/bin/bash

set -ex

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

KUBECONFIG="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig"
KUBECTL="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl --kubeconfig=${KUBECONFIG}"

MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/${KUBEVIRT_PROVIDER}/manifests"
MULTUS_MANIFEST="${MANIFESTS_DIR}/multus.yaml"

CUSTOM_MANIFESTS="${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/manifests"
SRIOV_COMPONENTS_MANIFEST="${CUSTOM_MANIFESTS}/sriov-components.yaml"
SRIOV_DEVICE_PLUGIN_CONFIG_MANIFEST="${CUSTOM_MANIFESTS}/sriovdp-config.yaml"

SRIOV_COMPONENTS_NAMESPACE="${SRIOV_COMPONENTS_NAMESPACE:-sriov}"

function _kubectl() {
    ${KUBECTL} "$@"
}

function _retry() {
  local -r tries=$1
  local -r wait_time=$2
  local -r action=$3
  local -r wait_message=$4
  local -r waiting_action=$5

  eval $action
  local return_code=$?
  for i in $(seq $tries); do
    if [[ $return_code -ne 0 ]]; then
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
    pods_not_ready_count=$(grep -cw False <<<"$all_pods_ready_condition")
    if [ "$pods_not_ready_count" -eq 0 ]; then
      return 0
    fi
  fi

  return 1
}

# not using kubectl wait since with the sriov operator the pods get restarted a couple of times and this is
# more reliable
function sriov_components::wait_pods_ready() {
  local -r tries=30
  local -r wait_time=10

  local -r wait_message="Waiting for all pods to become ready.."
  local -r error_message="Not all pods were ready after $(($tries * $wait_time)) seconds"

  local -r get_pods='_kubectl get pods --all-namespaces'
  local -r action="_check_all_pods_ready"

  set +x
  trap "set -x" RETURN

  if ! _retry "$tries" "$wait_time" "$action" "$wait_message" "$get_pods"; then
    echo $error_message
    return 1
  fi

  echo "all pods are ready"
  return 0
}

function sriov_components::wait_allocatable_resource() {
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

  if ! _retry $tries $wait_time "$action" "$wait_message"; then
    echo $error_message
    return 1
  fi

  return 0
}

function sriov_components::deploy_multus() {
  echo 'Deploying Multus'
  _kubectl apply -f "$MULTUS_MANIFEST"

  return 0
}

function sriov_components::deploy_sriov_components() {
  local -r pf_names=$1

  _create_custom_manifests_dir
  _prepare_device_plugin_config "$pf_names"
  _deploy_sriov_components

  return 0
}

function _create_custom_manifests_dir() {
  mkdir -p "$CUSTOM_MANIFESTS"

  cp -f "$MANIFESTS_DIR"/* "$CUSTOM_MANIFESTS"

  return 0
}

function _prepare_device_plugin_config() {
  local -r pf_names=$1

  # Format the input PF name to JSON string array
  local -r pf_names_json_array=$(_format_json_array "$pf_names")

  # Patch sriov-dp ConfigMap, pfNames with PF names json array
  sed -i "s?pfNames\":.*?pfNames\": $pf_names_json_array?g" "$SRIOV_DEVICE_PLUGIN_CONFIG_MANIFEST"

  return 0
}

function _format_json_array() {
  local -r string=$1

  local json_array="$string"
  # Replace all spaces with ",": aa bb -> aa","bb
  local -r replace='","'
  json_array="${json_array// /$replace}"

  # Add opening quotes for first element, and closing quotes for last element
  # aa","bb -> "aa","bb"
  json_array="\"${json_array}\""

  # Add brackets: "aa","bb" -> ["aa","bb"]
  json_array="[${json_array}]"

  echo "$json_array"
}

function _deploy_sriov_components() {
  _kubectl kustomize "$CUSTOM_MANIFESTS" >"$SRIOV_COMPONENTS_MANIFEST"

  echo "Deploying SRIOV components:"
  cat "$SRIOV_COMPONENTS_MANIFEST"

  _kubectl apply -f "$SRIOV_COMPONENTS_MANIFEST"

  return 0
}

function sriov_components::get_resource_name() {
  local resource_name
  resource_name=$(sed -n 's/.*"resourceName": *//p' "$SRIOV_COMPONENTS_MANIFEST")
  resource_name=$(sed 's/"//g' <<<"$resource_name")
  resource_name=$(sed 's/,//g' <<<"$resource_name")
  resource_name=$(sed 's/ //g' <<<"$resource_name")

  local resource_prefix
  resource_prefix=$(sed -n 's/.*resource-prefix= *//p' "$SRIOV_COMPONENTS_MANIFEST")
  resource_prefix=$(sed 's/ //g' <<<"$resource_prefix")

  echo "$resource_prefix/$resource_name"
}
