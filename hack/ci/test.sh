#!/usr/bin/env bash

set -euo pipefail

export PATH=$PATH:/usr/local/go/bin/
which kubectl
which oc
echo "checking nodes for cluster"
# in CI, this cmd fails unless you provide a ns
oc -n default get nodes
echo "checking configuration"
env | grep KUBE
kubectl config view
export DOCKER_PREFIX='dhiller'
export DOCKER_TAG="latest"
export KUBEVIRT_PROVIDER=external
export GIMME_GO_VERSION=1.12.8
export GOPATH="/go"
export GOBIN="/usr/bin"
source /etc/profile.d/gimme.sh
echo "checking configuration location"
echo "KUBECONFIG: ${KUBECONFIG}"
oc project default
echo "calling cluster-up to prepare config and check whether cluster is reachable"
bash -x ./cluster-up/up.sh
echo "checking cluster configuration after config prep"
kubectl config view
echo "deploying"
bash -x ./hack/cluster-deploy.sh
echo "checking pods for kubevirt"
oc get pods -n kubevirt
echo "testing"
bash -x ./hack/functests.sh
