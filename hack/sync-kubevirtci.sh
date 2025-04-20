#!/bin/bash
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright The KubeVirt Authors.
#


set -ex

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

# update cluster-up if needed
version_file="kubevirtci/cluster-up/version.txt"
sha_file="kubevirtci/cluster-up-sha.txt"
download_cluster_up=true
function getClusterUpShasum() {
    (
        cd ${KUBEVIRT_DIR}
        # We use LC_ALL=C to make sort canonical between machines, this is
        # from sort man page [1]:
        # ```
        # *** WARNING *** The locale specified by the environment affects sort
        # order.  Set LC_ALL=C to get the traditional sort order that uses
        # native byte values.
        # ```
        # [1] https://man7.org/linux/man-pages/man1/sort.1.html
        find kubevirtci/cluster-up -type f | LC_ALL=C sort | xargs sha1sum | sha1sum | awk '{print $1}'
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
    rm -rf kubevirtci/cluster-up
    (
        cd kubevirtci
        curl --fail -L https://github.com/kubevirt/kubevirtci/archive/refs/tags/${kubevirtci_git_hash}.tar.gz | tar xz kubevirtci-${kubevirtci_git_hash}/cluster-up --strip-component 1
    )

    echo ${kubevirtci_git_hash} >${version_file}
    new_sha=$(getClusterUpShasum)
    echo ${new_sha} >${sha_file}
    echo "KUBEVIRTCI_TAG=${kubevirtci_git_hash}" >>kubevirtci/cluster-up/hack/common.sh
fi
