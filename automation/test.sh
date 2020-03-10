#!/bin/bash -xe

if [[ $TARGET =~ okd-.* ]]; then
  export KUBEVIRT_PROVIDER="okd-4.1"
  export KUBEVIRT_MEMORY_SIZE=6144M
elif [[ $TARGET =~ k8s-.* ]]; then
  export KUBEVIRT_PROVIDER="k8s-1.17"
fi

export KUBEVIRT_NUM_NODES=2

kubectl() { cluster/kubectl.sh "$@"; }

make cluster-down
make cluster-up
make cluster-sync
make ci-functest

# Upgrade test requires OLM which is currently
# only available with okd providers
if [[ $TARGET =~ okd-.* ]]; then
  make upgrade-test
  make ci-functest
fi
