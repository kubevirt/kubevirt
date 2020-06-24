#!/bin/bash -xe

export KUBEVIRT_PROVIDER="$TARGET"

if [[ $TARGET =~ okd-.* || $TARGET =~ ocp-.* ]]; then
  export KUBEVIRT_MEMORY_SIZE=6144M
fi

export KUBEVIRT_NUM_NODES=2

kubectl() { cluster/kubectl.sh "$@"; }

make cluster-down
make cluster-up
make cluster-sync
make ci-functest

# Upgrade test requires OLM which is currently
# only available with okd providers
if [[ $TARGET =~ okd-.* || $TARGET =~ ocp-.* ]]; then
  make upgrade-test
  make ci-functest
fi
