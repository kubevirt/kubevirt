#!/bin/bash

set -ex

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

val=$(git ls-remote https://github.com/kubevirt/kubevirtci | grep HEAD | awk '{print $1}')
sed -i "/^[[:blank:]]*kubevirtci_git_hash[[:blank:]]*=/s/=.*/=\"${val}\"/" hack/config-default.sh

hack/sync-kubevirtci.sh
