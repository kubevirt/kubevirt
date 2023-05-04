#!/bin/bash -xe

export KUBEVIRT_PROVIDER="$TARGET"

export KUBEVIRT_MEMORY_SIZE=12G
export KUBEVIRT_NUM_NODES=4
export KUBEVIRT_DEPLOY_PROMETHEUS=true

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
