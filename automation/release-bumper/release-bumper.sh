#!/usr/bin/env bash

set -ex

CONFIG_FILE="hack/config"

function main {
  declare -A CURRENT_VERSIONS
  declare -A UPDATED_VERSIONS
  declare -A COMPONENTS_REPOS
  declare -A IMPORT_REPOS
  declare SHOULD_UPDATED
  declare TARGET_BRANCH
  declare UPDATE_TYPE

  get_current_branch

  echo "INFO: Checking go version"
  go version

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

  echo INFO: Updating image digest list...
  ./automation/digester/update_images.sh

  echo INFO: Updating go.mod...
  update_go_mod

  echo INFO: Executing "go mod tidy"
  go mod tidy -v

  echo INFO: Executing "go mod vendor"
  go mod vendor

  echo INFO: Executing "build-manifests.sh"...
  ./hack/build-manifests.sh
}

function get_current_branch() {
  if TARGET_BRANCH=$(git symbolic-ref --short -q HEAD)
  then
    echo on branch "$TARGET_BRANCH"
  else
    echo no branch is checked out
    exit 1
  fi

  if [ "$TARGET_BRANCH" == "main" ]
  then
    UPDATE_TYPE="all"
  else
    UPDATE_TYPE="z_release"
  fi
}

function get_current_versions {
  CURRENT_VERSIONS=(
    ["KUBEVIRT"]=""
    ["CDI"]=""
    ["NETWORK_ADDONS"]=""
    ["SSP"]=""
    ["TTO"]=""
    ["HPPO"]=""
    ["HPP"]=""
    ["KUBEVIRT_CONSOLE_PLUGIN"]=""
  )

  for component in "${!CURRENT_VERSIONS[@]}"; do
    CURRENT_VERSIONS[$component]=$(grep "$component"_VERSION ${CONFIG_FILE} | sed -r "s|${component}_VERSION=\"(.+)\"$|\1|")
    done;
}

function get_updated_versions {
  COMPONENTS_REPOS=(
    ["KUBEVIRT"]="kubevirt/kubevirt"
    ["CDI"]="kubevirt/containerized-data-importer"
    ["NETWORK_ADDONS"]="kubevirt/cluster-network-addons-operator"
    ["SSP"]="kubevirt/ssp-operator"
    ["TTO"]="kubevirt/tekton-tasks-operator"
    ["HPPO"]="kubevirt/hostpath-provisioner-operator"
    ["HPP"]="kubevirt/hostpath-provisioner"
    ["KUBEVIRT_CONSOLE_PLUGIN"]="kubevirt-ui/kubevirt-plugin"
  )

  IMPORT_REPOS=(
    ["KUBEVIRT"]="kubevirt.io/api"
    ["CDI"]="kubevirt.io/containerized-data-importer-api"
    ["NETWORK_ADDONS"]="kubevirt/cluster-network-addons-operator"
    ["SSP"]="kubevirt.io/ssp-operator/api"
    ["TTO"]="kubevirt/tekton-tasks-operator/api"
  )

  UPDATED_VERSIONS=()
  if [[ -n ${UPDATED_COMPONENT} ]]; then
    if [[ -z ${UPDATED_VERSION} ]]; then
      UPDATED_VERSION=$(get_latest_release "$UPDATED_COMPONENT")
    fi
    if [[ -v COMPONENTS_REPOS[${UPDATED_COMPONENT}] ]]; then
      HTTP_CODE=$(curl "https://api.github.com/repos/${COMPONENTS_REPOS[$UPDATED_COMPONENT]}/releases/tags/${UPDATED_VERSION}" --write-out '%{http_code}' --silent --output /dev/null)
      if [[ ${HTTP_CODE} == "200" ]]; then
        UPDATED_VERSIONS["${UPDATED_COMPONENT}"]="${UPDATED_VERSION}"
      else
        echo "ERROR: unknown version '${UPDATED_VERSION}' for component '${UPDATED_COMPONENT}'"
        exit 1
      fi
    else
      echo "ERROR: unknown component '${UPDATED_COMPONENT}'"
      exit 1
    fi
  else
    for component in "${!COMPONENTS_REPOS[@]}"; do
      UPDATED_VERSIONS[$component]=$(get_latest_release "$component");
      if [ -z "${UPDATED_VERSIONS[$component]}" ]; then
        echo "ERROR: Unable to get an updated version of $component, aborting..."
        exit 1
      fi
    done;
  fi
}

function get_latest_release() {
  repo="${COMPONENTS_REPOS[$1]}"
  current_version="${CURRENT_VERSIONS[$1]}"

  major=$(echo $current_version | cut -d. -f1)
  minor=$(echo $current_version | cut -d. -f2)

  RELEASES=$(curl -s -L "https://api.github.com/repos/$repo/releases" | jq -r '.[].tag_name')
  releases=(${RELEASES})

  semversort "${releases[*]}"

  for (( i=${#KEYS_ARR[@]}-1 ; i >= 0 ; i-- )) ; do
    release=${releases[${KEYS_ARR[$i]}]}

    new_major=$(echo $release | cut -d. -f1)
    new_minor=$(echo $release | cut -d. -f2)

    if [ "$UPDATE_TYPE" = "all" ]; then
      break;
    elif [ "$UPDATE_TYPE" = "z_release" ] && [ "$major" = "$new_major" ] && [ "$minor" = "$new_minor" ]; then
      break;
    fi
  done

  echo "${release}"
}

function compare_versions() {
  # comparing between current (local) components versions and their counterparts in the remote repositories.
  for component in "${!UPDATED_VERSIONS[@]}"; do
    versions=("${CURRENT_VERSIONS[$component]}" "${UPDATED_VERSIONS[$component]}")
    semversort "${versions[*]}"

    if [ ${CURRENT_VERSIONS[$component]} != ${UPDATED_VERSIONS[$component]} ] \
     && [ "${versions[${KEYS_ARR[-1]}]}" == ${UPDATED_VERSIONS[$component]} ]; then
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

  KEYS_ARR=(${keys})
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
  PR=$(curl -s -L https://api.github.com/repos/kubevirt/hyperconverged-cluster-operator/pulls | jq "[.[] | {title: .title, ref: .base.ref}]" )

  for component in "${SHOULD_UPDATED[@]}"; do
    echo INFO: Checking update for "$component";

    # Check if pull request for that component and version already exists
    search_pattern=$(echo "$component.*${UPDATED_VERSIONS[$component]}" | tr -d '"')

    search_pr=$(jq "[.[] | select((.title | test(\"${search_pattern}\")) and (.ref == \"${TARGET_BRANCH}\"))] | length" <<< "$PR")

    if [[ $search_pr -ne 0 ]] ; then
      echo "INFO: An existing pull request for bumping $component to version ${UPDATED_VERSIONS[$component]} has been found. \
Continuing to next component."
      continue
    else
      echo "INFO: Updating $component to ${UPDATED_VERSIONS[$component]}."
      sed -E -i "s|(${component}_VERSION=).*|\1\"${UPDATED_VERSIONS[$component]}\"|" ${CONFIG_FILE}
      echo "$component" > updated_component.txt
      echo "${UPDATED_VERSIONS[$component]}" > updated_version.txt
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

  if [[ -v IMPORT_REPOS[$UPDATED_COMPONENT] ]]; then
    MODULE_PATH=${IMPORT_REPOS[$UPDATED_COMPONENT]}
    sed -E -i "s|(${MODULE_PATH}.*)v.+|\1${UPDATED_VERSION}|" go.mod
  else
    echo "No need to update go.mod for ${UPDATED_COMPONENT}"
  fi

}

main
