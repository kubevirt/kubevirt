#!/bin/bash

CONFIG_FILE="hack/config"

function main {
  declare -A CURRENT_VERSIONS
  declare -A UPDATED_VERSIONS
  declare -A COMPONENTS_REPOS
  declare SHOULD_UPDATED

  echo "INFO: Getting Components current versions..."
  get_current_versions

  echo "INFO: Getting Components updated versions..."
  get_updated_versions

  echo "INFO: Comparing Versions..."
  compare_versions

  if [ ${#SHOULD_UPDATED[@]} == 0 ]; then
    echo "INFO: All components are already in latest version. Job is completed.";
    exit 0;
  fi

  update_versions

  echo INFO: Executing "build-manifests.sh"...
  ./hack/build-manifests.sh

  echo INFO: Updating go.mod...
  update_go_mod

  echo INFO: Executing "go mod vendor"
  go mod vendor

  echo INFO: Executing "go mod tidy"
  go mod tidy
}

function get_current_versions {
  CURRENT_VERSIONS=(
    ["KUBEVIRT"]=""
    ["CDI"]=""
    ["NETWORK_ADDONS"]=""
    ["SSP"]=""
    ["NMO"]=""
    ["HPPO"]=""
    ["HPP"]=""
    ["VM_IMPORT"]=""
#    ["CONVERSION_CONTAINER"]=""
#    ["VMWARE_CONTAINER"]=""
  )

  for component in "${!CURRENT_VERSIONS[@]}"; do
    CURRENT_VERSIONS[$component]=$(grep "$component"_VERSION ${CONFIG_FILE} | cut -d "=" -f 2)
    done;
}

function get_updated_versions {
  COMPONENTS_REPOS=(
    ["KUBEVIRT"]="kubevirt/kubevirt"
    ["CDI"]="kubevirt/containerized-data-importer"
    ["NETWORK_ADDONS"]="kubevirt/cluster-network-addons-operator"
    ["SSP"]="kubevirt/kubevirt-ssp-operator"
    ["NMO"]="kubevirt/node-maintenance-operator"
    ["HPPO"]="kubevirt/hostpath-provisioner-operator"
    ["HPP"]="kubevirt/hostpath-provisioner"
    ["VM_IMPORT"]="kubevirt/vm-import-operator"
#    ["CONVERSION_CONTAINER"]=""
#    ["VMWARE_CONTAINER"]=""
  )

  UPDATED_VERSIONS=()
  for component in "${!COMPONENTS_REPOS[@]}"; do
    UPDATED_VERSIONS[$component]=\"$(get_latest_release "${COMPONENTS_REPOS[$component]}")\";
    if [ "${UPDATED_VERSIONS[$component]}" == \"\" ]; then
      echo "ERROR: Unable to get an updated version of $component, aborting..."
      exit 1
    fi
    done;
}

function get_latest_release() {
  curl -s -L --silent "https://api.github.com/repos/$1/releases" | grep -m 1 '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

function compare_versions() {
  # comparing between current (local) components versions and their counterparts in the remote repositories.
  for component in "${!UPDATED_VERSIONS[@]}"; do
    higher_version=$(semversort ${CURRENT_VERSIONS[$component]} ${UPDATED_VERSIONS[$component]});
    if [ ${CURRENT_VERSIONS[$component]} != ${UPDATED_VERSIONS[$component]} ] \
     && [ ${higher_version} == ${UPDATED_VERSIONS[$component]} ]; then
      echo "INFO: $component" is outdated. Current: "${CURRENT_VERSIONS[$component]}", Updated: "${UPDATED_VERSIONS[$component]}"
      SHOULD_UPDATED+=( "$component" )
    fi
  done;
}

function semversort() {
  versions_list=$@

  tags_orig=(${versions_list})
  tags_weight=($(version_weight "${tags_orig[*]}"))

  keys=$(for ix in ${!tags_weight[*]}; do
    printf "%s+%s\n" "${tags_weight[${ix}]}" ${ix}
  done | sort -V | cut -d+ -f2)

  keys_arr=(${keys})
  echo ${tags_orig[${keys_arr[-1]}]}
}

function version_weight() {
  echo -e "$1" | tr ' ' "\n" | sed -e 's:\+.*$::' | sed -e 's:^v::' |
    sed -re 's:^[0-9]+(\.[0-9]+)+$:&-stable:' |
    sed -re 's:([^A-Za-z])dev\.?([^A-Za-z]|$):\1.10.\2:g' |
    sed -re 's:([^A-Za-z])(alpha|a)\.?([^A-Za-z]|$):\1.20.\3:g' |
    sed -re 's:([^A-Za-z])(beta|b)\.?([^A-Za-z]|$):\1.30.\3:g' |
    sed -re 's:([^A-Za-z])(rc|RC)\.?([^A-Za-z]|$)?:\1.40.\3:g' |
    sed -re 's:([^A-Za-z])stable\.?([^A-Za-z]|$):\1.50.\2:g' |
    sed -re 's:([^A-Za-z])pl\.?([^A-Za-z]|$):\1.60.\2:g' |
    sed -re 's:([^A-Za-z])(patch|p)\.?([^A-Za-z]|$):\1.70.\3:g' |
    sed -r 's:\.{2,}:.:' |
    sed -r 's:\.$::' |
    sed -r 's:-\.:.:'
}

function update_versions() {
  for component in "${SHOULD_UPDATED[@]}"; do
    echo INFO: Checking update for "$component";

    # Check if pull request for that component and version already exists
    search_pattern=$(echo "$component.*${UPDATED_VERSIONS[$component]}" | tr -d '"')
    if curl -s -L  https://api.github.com/repos/kubevirt/hyperconverged-cluster-operator/pulls | jq .[].title | \
    grep -q "$search_pattern"; then
      echo "INFO: An existing pull request for bumping $component to version ${UPDATED_VERSIONS[$component]} has been found. \
Continuing to next component."
      continue
    else
      echo "INFO: Updating $component to ${UPDATED_VERSIONS[$component]}."
      sed -E -i "s/(""$component""_VERSION=).*/\1${UPDATED_VERSIONS[$component]}/" ${CONFIG_FILE}

      echo "$component" > updated_component.txt
      echo "${UPDATED_VERSIONS[$component]}" | tr -d '"' > updated_version.txt
      UPDATING='true'
      break
    fi
  done;

  if [ "${UPDATING}" != 'true' ]; then
    echo "INFO: There are no more components to update. Finishing Job Successfully."
    exit 0
  fi
}

function update_go_mod() {
  UPDATED_COMPONENT=$(cat updated_component.txt)
  UPDATED_VERSION=$(cat updated_version.txt)

  if [ $UPDATED_COMPONENT == "KUBEVIRT" ]; then
    MODULE_PATH="kubevirt.io"
    EXCLUSION="/containerized-data-importer/!"
  else
    MODULE_PATH=$(echo ${COMPONENTS_REPOS[$UPDATED_COMPONENT]} | cut -d "/" -f 2)
  fi

  sed -E -i "$EXCLUSION s/($MODULE_PATH.*)v.+/\1${UPDATED_VERSION}/" go.mod
}

main
