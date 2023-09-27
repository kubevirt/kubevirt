#!/bin/bash

set -ex

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

function latest_version() {
    local org="$1"
    local repo="$2"

    curl --fail -s "https://api.github.com/repos/${org}/${repo}/releases/latest" |
        jq -r '.tag_name'
}

function checksum() {
    local version="$1"
    local file="$2"

    curl -L "https://github.com/kubevirt/common-instancetypes/releases/download/${version}/CHECKSUMS.sha256" |
        grep "${file}" | cut -d " " -f 1
}

version=$(latest_version "kubevirt" "common-instancetypes")
instancetypes_checksum=$(checksum "${version}" "common-clusterinstancetypes-bundle-${version}.yaml")
preferences_checksum=$(checksum "${version}" "common-clusterpreferences-bundle-${version}.yaml")

sed -i "/^[[:blank:]]*common_instancetypes_version[[:blank:]]*=/s/=.*/=\${COMMON_INSTANCETYPES_VERSION:-\"${version}\"}/" hack/config-default.sh
sed -i "/^[[:blank:]]*cluster_instancetypes_sha256[[:blank:]]*=/s/=.*/=\${CLUSTER_INSTANCETYPES_SHA256:-\"${instancetypes_checksum}\"}/" hack/config-default.sh
sed -i "/^[[:blank:]]*cluster_preferences_sha256[[:blank:]]*=/s/=.*/=\${CLUSTER_PREFERENCES_SHA256:-\"${preferences_checksum}\"}/" hack/config-default.sh

hack/sync-common-instancetypes.sh
