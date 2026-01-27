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

# Get image digest from quay.io for a specific architecture
function image_digest() {
    local image="$1"
    local tag="$2"
    local arch="$3"
    skopeo inspect --raw "docker://quay.io/kubevirt/${image}:${tag}" |
        jq -r ".manifests[] | select(.platform.architecture == \"${arch}\") | .digest"
}

version=$(latest_version)
yaml_checksum=$(checksum "${version}" "install-virt-operator.yaml")

# Get image digests for all architectures
apiserver_amd64=$(image_digest "virt-template-apiserver" "${version}" "amd64")
apiserver_arm64=$(image_digest "virt-template-apiserver" "${version}" "arm64")
apiserver_s390x=$(image_digest "virt-template-apiserver" "${version}" "s390x")

controller_amd64=$(image_digest "virt-template-controller" "${version}" "amd64")
controller_arm64=$(image_digest "virt-template-controller" "${version}" "arm64")
controller_s390x=$(image_digest "virt-template-controller" "${version}" "s390x")

# Update default.sh with version and yaml checksum
sed -i "/^[[:blank:]]*virt_template_version[[:blank:]]*=/s/=.*/=\${VIRT_TEMPLATE_VERSION:-\"${version}\"}/" "$(dirname "$0")/default.sh"
sed -i "/^[[:blank:]]*virt_template_yaml_sha256[[:blank:]]*=/s/=.*/=\${VIRT_TEMPLATE_YAML_SHA256:-\"${yaml_checksum}\"}/" "$(dirname "$0")/default.sh"

# Update deps.bzl with new digests
deps_file="$(dirname "$0")/../../images/virt-template/deps.bzl"
sed -i "s|^VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = .*|VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = \"${apiserver_amd64}\"|" "${deps_file}"
sed -i "s|^VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = .*|VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = \"${apiserver_arm64}\"|" "${deps_file}"
sed -i "s|^VIRT_TEMPLATE_APISERVER_DIGEST_S390X = .*|VIRT_TEMPLATE_APISERVER_DIGEST_S390X = \"${apiserver_s390x}\"|" "${deps_file}"

sed -i "s|^VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = .*|VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = \"${controller_amd64}\"|" "${deps_file}"
sed -i "s|^VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = .*|VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = \"${controller_arm64}\"|" "${deps_file}"
sed -i "s|^VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = .*|VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = \"${controller_s390x}\"|" "${deps_file}"

"$(dirname "$0")/sync.sh"
