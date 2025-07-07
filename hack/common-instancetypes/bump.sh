#!/bin/bash

set -ex

TARGET_BRANCH=${1:-"main"}

function latest_version() {
    curl --fail -s "https://api.github.com/repos/kubevirt/common-instancetypes/releases?per_page=100" |
        jq -r '.[] | select(.target_commitish == '\""${TARGET_BRANCH}"\"') | .tag_name' | head -n1
}

function checksum() {
    local version="$1"
    local file="$2"

    curl -L "https://github.com/kubevirt/common-instancetypes/releases/download/${version}/CHECKSUMS.sha256" |
        grep "${file}" | cut -d " " -f 1
}

version=$(latest_version)
instancetypes_checksum=$(checksum "${version}" "common-clusterinstancetypes-bundle-${version}.yaml")
preferences_checksum=$(checksum "${version}" "common-clusterpreferences-bundle-${version}.yaml")

sed -i "/^[[:blank:]]*common_instancetypes_version[[:blank:]]*=/s/=.*/=\${COMMON_INSTANCETYPES_VERSION:-\"${version}\"}/" $(dirname "$0")/default.sh
sed -i "/^[[:blank:]]*cluster_instancetypes_sha256[[:blank:]]*=/s/=.*/=\${CLUSTER_INSTANCETYPES_SHA256:-\"${instancetypes_checksum}\"}/" $(dirname "$0")/default.sh
sed -i "/^[[:blank:]]*cluster_preferences_sha256[[:blank:]]*=/s/=.*/=\${CLUSTER_PREFERENCES_SHA256:-\"${preferences_checksum}\"}/" $(dirname "$0")/default.sh

$(dirname "$0")/sync.sh
