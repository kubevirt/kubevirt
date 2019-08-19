#!/bin/bash -xe

if [[ $TARGET =~ okd-.* ]]; then
  export KUBEVIRT_PROVIDER="okd-4.1.0"
  export KUBEVIRT_MEMORY_SIZE=6144M
elif [[ $TARGET =~ k8s-.* ]]; then
  export KUBEVIRT_PROVIDER="k8s-1.15.1"
fi

export KUBEVIRT_NUM_NODES=2

kubectl() { cluster-up/kubectl.sh "$@"; }

make cluster-down
make cluster-up
make cluster-sync
CMD='cluster-up/kubectl.sh' make functest
