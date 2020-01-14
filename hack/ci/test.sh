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
# enable nested-virt
oc get machineset -n openshift-machine-api -o json >/tmp/machinesets.json
MACHINE_IMAGE=$(jq .items[0].spec.template.spec.providerSpec.value.disks[0].image /tmp/machinesets.json)
NESTED_VIRT_IMAGE="sotest-rhcos-nested-virt"
sed -i "s/$MACHINE_IMAGE/$NESTED_VIRT_IMAGE/g" /tmp/machinesets.json
sed -i 's/sotest-rhcos-nested-virt/"sotest-rhcos-nested-virt"/g' /tmp/machinesets.json
oc apply -f /tmp/machinesets.json
oc scale --replicas=0 machineset --all -n openshift-machine-api
oc get machines -n openshift-machine-api -o json >/tmp/machines.json
num_machines=$(jq '.items | length' /tmp/machines.json)
while [ "$num_machines" -ne "3" ]; do
    oc get machines -n openshift-machine-api -o json >/tmp/machines.json
    num_machines=$(jq '.items | length' /tmp/machines.json)
done
oc scale --replicas=1 machineset --all -n openshift-machine-api
while [ "$num_machines" -ne "6" ]; do
    oc get machines -n openshift-machine-api -o json >/tmp/machines.json
    num_machines=$(jq '.items | length' /tmp/machines.json)
done
while [ $(oc get nodes | wc -l) -ne "7" ]; do oc get nodes; done
nodes_ready=false
while ! "$nodes_ready"; do sleep 5 && if ! oc get nodes | grep NotReady; then nodes_ready=true; fi; done
oc project default
oc apply -f /go/src/kubevirt.io/kubevirt/kvm-ds.yml
workers=$(oc get nodes | grep worker | awk '{ print $1 }')
workers_each=($workers)
for i in {0..2}; do
    if ! oc debug node/"${workers_each[i]}" -- ls /dev/kvm; then oc debug node/"${workers_each[i]}" -- ls /dev/kvm; fi
done
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
