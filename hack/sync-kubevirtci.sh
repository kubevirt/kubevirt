#!/bin/bash

set -ex

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

# update cluster-up if needed
version_file="cluster-up/version.txt"
sha_file="cluster-up-sha.txt"
download_cluster_up=true
function getClusterUpShasum() {
    (
        cd ${KUBEVIRT_DIR}
        find cluster-up -type f | sort | xargs sha1sum | sha1sum | awk '{print $1}'
    )
}

# check if we got a new cluster-up git commit hash
if [[ -f "${version_file}" ]] && [[ $(cat ${version_file}) == ${kubevirtci_git_hash} ]]; then
    # check if files are modified
    current_sha=$(getClusterUpShasum)
    if [[ -f "${sha_file}" ]] && [[ $(cat ${sha_file}) == ${current_sha} ]]; then
        echo "cluster-up is up to date and not modified"
        download_cluster_up=false
    else
        echo "cluster-up was modified"
    fi
else
    echo "cluster-up git commit hash was updated"
fi
if [[ "$download_cluster_up" == true ]]; then
    echo "downloading cluster-up"
    rm -rf cluster-up
    curl -L https://github.com/kubevirt/kubevirtci/archive/${kubevirtci_git_hash}/kubevirtci.tar.gz | tar xz kubevirtci-${kubevirtci_git_hash}/cluster-up --strip-component 1

    echo ${kubevirtci_git_hash} >${version_file}
    new_sha=$(getClusterUpShasum)
    echo ${new_sha} >${sha_file}
fi
