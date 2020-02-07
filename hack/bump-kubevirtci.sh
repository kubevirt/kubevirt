#!/bin/bash
val=$(git ls-remote https://github.com/kubevirt/kubevirtci | grep HEAD | awk '{print $1}')
sed -i "/^[[:blank:]]*kubevirtci_git_hash[[:blank:]]*=/s/=.*/=\"${val}\"/" hack/config-default.sh
git --no-pager diff hack/config-default.sh | grep "kubevirtci_git_hash"
