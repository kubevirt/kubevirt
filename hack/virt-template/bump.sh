#!/bin/bash

set -ex

TARGET_BRANCH=${1:-"main"}

# Fetch latest release version from GitHub API
function latest_version() {
    curl --fail -s "https://api.github.com/repos/kubevirt/virt-template/releases?per_page=100" |
        jq -r '.[] | select(.target_commitish == '\""${TARGET_BRANCH}"\"') | .tag_name' | head -n1
}

# Get checksum for YAML manifest from release
function checksum() {
    local version="$1"
    local file="$2"
    curl -L "https://github.com/kubevirt/virt-template/releases/download/${version}/CHECKSUMS.sha256" |
        grep "${file}" | cut -d " " -f 1
}

# Get the multi-arch manifest list digest mirrored by push-images.sh
function list_digest() {
    local image="$1"
    local tag="$2"
    skopeo inspect --format '{{.Digest}}' "docker://quay.io/kubevirt/${image}:${tag}"
}

version=$(latest_version)
yaml_checksum=$(checksum "${version}" "install-virt-operator.yaml")
apiserver_list=$(list_digest "virt-template-apiserver" "${version}")
controller_list=$(list_digest "virt-template-controller" "${version}")

# Update default.sh with version, yaml checksum, and manifest list digests
sed -i "/^[[:blank:]]*virt_template_version[[:blank:]]*=/s/=.*/=\${VIRT_TEMPLATE_VERSION:-\"${version}\"}/" "$(dirname "$0")/default.sh"
sed -i "/^[[:blank:]]*virt_template_yaml_sha256[[:blank:]]*=/s/=.*/=\${VIRT_TEMPLATE_YAML_SHA256:-\"${yaml_checksum}\"}/" "$(dirname "$0")/default.sh"
sed -i "/^[[:blank:]]*virt_template_apiserver_digest[[:blank:]]*=/s/=.*/=\${VIRT_TEMPLATE_APISERVER_DIGEST:-\"${apiserver_list}\"}/" "$(dirname "$0")/default.sh"
sed -i "/^[[:blank:]]*virt_template_controller_digest[[:blank:]]*=/s/=.*/=\${VIRT_TEMPLATE_CONTROLLER_DIGEST:-\"${controller_list}\"}/" "$(dirname "$0")/default.sh"

"$(dirname "$0")/sync.sh"
