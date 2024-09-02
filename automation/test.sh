#!/bin/bash -xe

export KUBEVIRT_PROVIDER="$TARGET"

export KUBEVIRT_MEMORY_SIZE=12G
export KUBEVIRT_NUM_NODES=4
export KUBEVIRT_DEPLOY_PROMETHEUS=true

kubectl() { cluster/kubectl.sh "$@"; }

make cluster-down
make cluster-up

trap '{ make cluster-down; }' EXIT SIGINT SIGTERM SIGSTOP

make cluster-sync
export KUBECONFIG=$(_kubevirtci/cluster-up/kubeconfig.sh)
JOB_TYPE="stdci" GINKGO_LABELS=${GINKGO_LABELS} make functest
